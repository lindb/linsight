// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ljson "github.com/lindb/arrow/pkg/json"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// maxQueryRows is the maximum number of rows returned to the LLM to avoid
// exhausting the context window.
const maxQueryRows = 500

// datasourceQueryInput is the input struct for linsight_datasource_query.
// Field descriptions become part of the tool's JSON schema sent to the LLM.
type datasourceQueryInput struct {
	DatasourceUID string `json:"datasource_uid" jsonschema:"required,description=Datasource UID (short alphanumeric string\\, e.g. 'uHI9ecwNz'). Call linsight_list_datasources first to resolve the UID for a named datasource."`
	Params        string `json:"params"         jsonschema:"required,description=JSON-encoded query parameters containing the SQL query. Example: {\\\"sql\\\":\\\"select node\\, used/total\\, timestamp from lindb.monitor.system.mem_stat\\\"}."`
	StartTime     string `json:"start_time"     jsonschema:"description=Query start time: Unix millisecond timestamp or relative string (now-1h\\, now-30m\\, now-1d). Default: now-1h."`
	EndTime       string `json:"end_time"       jsonschema:"description=Query end time: Unix millisecond timestamp or relative string. Default: now."`
}

// queryResult is the response shape of linsight_datasource_query.
type queryResult struct {
	DatasourceType string                   `json:"datasource_type"`
	Data           []map[string]interface{} `json:"data,omitempty"`
	RowCount       int                      `json:"row_count,omitempty"`
	Message        string                   `json:"message,omitempty"`
}

// queryDatasource executes a data query using the plugin layer.
func queryDatasource(
	ctx context.Context,
	srv datasourceReader,
	mgr pluginManager,
	input datasourceQueryInput,
) (*queryResult, error) {
	toolsLog.Info("tool: datasource_query called",
		logger.String("datasource_uid", input.DatasourceUID),
		logger.String("params", truncateStr(input.Params, 200)),
	)

	// Try to resolve data source UID if it looks like a name
	datasourceUID := input.DatasourceUID
	if !isValidUID(datasourceUID) {
		// If it doesn't look like a UID, try to find by name
		datasources, err := srv.GetDatasources(ctx)
		if err == nil {
			for _, ds := range datasources {
				if strings.EqualFold(ds.Name, datasourceUID) {
					datasourceUID = ds.UID
					break
				}
			}
		}
	}

	ds, err := srv.GetDatasourceByUID(ctx, datasourceUID)
	if err != nil {
		toolsLog.Error("tool: datasource_query failed: datasource not found",
			logger.String("datasource_uid", datasourceUID),
			logger.Error(err),
		)
		return nil, fmt.Errorf("datasource not found (uid=%s): %w", datasourceUID, err)
	}

	p, err := mgr.GetPlugin(ds)
	if err != nil {
		toolsLog.Error("tool: datasource_query failed: plugin unavailable",
			logger.String("datasource_type", ds.Type),
			logger.Error(err),
		)
		return nil, fmt.Errorf("datasource plugin unavailable (type=%s): %w", ds.Type, err)
	}

	timeRange, err := parseTimeRange(input.StartTime, input.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid time range: %w", err)
	}

	query := &model.Query{
		Datasource: model.TargetDatasource{UID: input.DatasourceUID},
		Params:     json.RawMessage(input.Params),
	}

	rs, err := p.DataQuery(ctx, query, timeRange)
	if err != nil {
		toolsLog.Error("tool: datasource_query failed: data query error",
			logger.String("datasource_uid", datasourceUID),
			logger.Error(err),
		)
		return nil, fmt.Errorf("data query failed: %w", err)
	}

	result := &queryResult{DatasourceType: ds.Type}

	if rs == nil {
		// LinGo DataQuery returns nil — known limitation; return helpful message.
		result.Message = fmt.Sprintf(
			"datasource type %q returned no data. This may be a known limitation of the plugin.", ds.Type)
		toolsLog.Info("tool: datasource_query done (nil result set)",
			logger.String("datasource_uid", datasourceUID),
		)
		return result, nil
	}

	rows, truncated, err := convertResultSet(rs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}
	result.Data = rows
	result.RowCount = len(rows)
	if truncated {
		result.Message = fmt.Sprintf(
			"result truncated to %d rows (result set exceeded the %d-row limit)", maxQueryRows, maxQueryRows)
	}
	toolsLog.Info("tool: datasource_query done",
		logger.String("datasource_uid", datasourceUID),
		logger.Int("row_count", result.RowCount),
	)
	return result, nil
}

// isValidUID checks if a string looks like a valid UUID
func isValidUID(s string) bool {
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) == 36 && strings.Count(s, "-") == 4 {
		return true
	}

	// Short UID format (common in LinSight)
	if len(s) >= 8 && len(s) <= 32 {
		// Check if it contains only alphanumeric characters and hyphens
		for _, r := range s {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-') {
				return false
			}
		}
		return true
	}

	return false
}

// parseTimeRange builds a model.TimeRange from optional start/end strings.
// Accepts Unix millisecond timestamps or relative expressions (now-1h, now, etc.).
// Defaults: start → "now-1h", end → "now".
func parseTimeRange(start, end string) (model.TimeRange, error) {
	if start == "" {
		start = "now-1h"
	}
	if end == "" {
		end = "now"
	}
	// Leverage model.TimeRange.UnmarshalJSON which handles both numeric
	// timestamps and relative time expressions.
	raw := fmt.Sprintf(`{"start":%q,"end":%q}`, start, end)
	var tr model.TimeRange
	if err := json.Unmarshal([]byte(raw), &tr); err != nil {
		return model.TimeRange{}, err
	}
	return tr, nil
}

// convertResultSet converts an Arrow ResultSet into a slice of column→value maps.
// At most maxQueryRows rows are returned; the second return value is true when
// the result was truncated.
func convertResultSet(rs *ljson.ResultSet) ([]map[string]interface{}, bool, error) {
	if rs == nil || rs.Schema == nil || len(rs.Rows) == 0 {
		return nil, false, nil
	}
	cols := rs.Schema.Columns
	rows := rs.Rows
	truncated := false
	if len(rows) > maxQueryRows {
		rows = rows[:maxQueryRows]
		truncated = true
	}

	result := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			if i < len(row) {
				m[col.Name] = row[i]
			}
		}
		result = append(result, m)
	}
	return result, truncated, nil
}
