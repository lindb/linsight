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

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/lindb/common/pkg/logger"

	toolscomp "github.com/lindb/linsight/ai/components/tools"
)

func init() {
	register(func(deps BuiltinDeps) (tool.BaseTool, error) {
		t, err := toolutils.InferTool(
			"linsight_list_datasources",
			"List all data sources accessible to the current user. "+
				"Call this first to discover available data sources before querying data.",
			func(ctx context.Context, _ struct{}) (*toolscomp.ToolCallResult, error) {
				return listDatasources(ctx, deps.DatasourceSrv)
			},
		)
		if err != nil {
			return nil, err
		}
		return withExtraInfo(t, map[string]any{
			"icon":        "datasource",
			"render_type": "list",
		}), nil
	})
}

// datasourceInfo is the per-datasource summary returned by linsight_list_datasources.
type datasourceInfo struct {
	UID       string `json:"uid"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Category  string `json:"category"`
	IsDefault bool   `json:"is_default"`
}

// listDatasources calls the service and maps the result to the LLM-friendly shape.
func listDatasources(ctx context.Context, srv datasourceReader) (*toolscomp.ToolCallResult, error) {
	toolsLog.Info("tool: list_datasources called")
	datasources, err := srv.GetDatasources(ctx)
	if err != nil {
		toolsLog.Error("tool: list_datasources failed", logger.Error(err))
		return nil, err
	}
	result := make([]datasourceInfo, 0, len(datasources))
	for _, ds := range datasources {
		result = append(result, datasourceInfo{
			UID:       ds.UID,
			Name:      ds.Name,
			Type:      ds.Type,
			Category:  ds.Category,
			IsDefault: ds.IsDefault,
		})
	}
	toolsLog.Info("tool: list_datasources done", logger.Int("count", len(result)))

	dataJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal datasource list: %w", err)
	}
	return &toolscomp.ToolCallResult{
		Data:  dataJSON,
		Extra: map[string]any{"count": len(result)},
	}, nil
}
