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

package schedule

import (
	"context"
	"sort"
	"time"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	ljson "github.com/lindb/arrow/pkg/json"
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// AlertRule wraps a model.AlertRule with compiled conditions and cached query parameters.
type AlertRule struct {
	conditions  []*vm.Program
	severities  []model.Severity // parallel to conditions; tracks which severity each program corresponds to
	queryParams *model.QueryRequest
	rule        *model.AlertRule
	querier     Querier
	stateMgr    RuleStateManager

	deps           *Deps
	currentVersion int
	ruleLogger     logger.Logger
}

// NewAlertRule creates an AlertRule wrapper.
func NewAlertRule(rule *model.AlertRule, deps *Deps) *AlertRule {
	return &AlertRule{
		rule:           rule,
		deps:           deps,
		querier:        NewQuerier(rule, deps),
		stateMgr:       NewRuleStateManager(),
		currentVersion: -1,
		ruleLogger:     logger.GetLogger("Alerting", "AlertRule"),
	}
}

// Update replaces the underlying rule model (triggered on version change).
func (r *AlertRule) Update(rule *model.AlertRule) {
	r.rule = rule
}

// Perpare compiles conditions when the rule version has changed.
func (r *AlertRule) Perpare() error {
	rule := r.rule
	if r.currentVersion == rule.Version {
		// same version — nothing to recompile
		return nil
	}

	conditions, err := rule.ToConditions()
	if err != nil {
		return err
	}
	dataQuery, err := rule.ToDataQuery()
	if err != nil {
		return err
	}
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].Severity < conditions[j].Severity
	})

	r.queryParams = dataQuery

	if len(conditions) == 0 {
		// No conditions configured — treat every series as a match (passthrough).
		// Compile a synthetic "always-true" program to reuse the existing Eval path.
		passthrough, err := expr.Compile("true")
		if err != nil {
			return err
		}
		r.conditions = []*vm.Program{passthrough}
		r.severities = []model.Severity{model.AlertSeverity}
		r.ruleLogger.Info("conditions compiled, passthrough enabled (no conditions configured)",
			logger.String("rule", rule.UID),
			logger.Int32("version", int32(rule.Version)))
	} else {
		r.conditions = make([]*vm.Program, len(conditions))
		r.severities = make([]model.Severity, len(conditions))

		for idx := range conditions {
			condition := conditions[idx]
			r.severities[idx] = condition.Severity
			var str string
			if err := encoding.JSONUnmarshal(condition.Expr, &str); err != nil {
				return err
			}
			r.ruleLogger.Info("condition compiled",
				logger.String("rule", rule.UID),
				logger.Int32("version", int32(rule.Version)),
				logger.Int32("severity", int32(condition.Severity)),
				logger.String("expr", str))
			program, err := expr.Compile(str)
			if err != nil {
				return err
			}
			r.conditions[idx] = program
		}
	}

	r.currentVersion = rule.Version
	return nil
}

// FetchData queries the datasource and returns a flat list of DataPoints.
// The time range is re-resolved on every call so relative expressions like "now-5m"
// always reflect the current wall clock, not the time Perpare() last ran.
//
// Each row in the Arrow ResultSet is processed as follows:
//   - "string" columns → DataPoint.Labels (GROUP-BY dimensions)
//   - "time_series" columns → one DataPoint per time point (existing per-point timestamps preserved)
//   - numeric scalar columns → merged into a single DataPoint.Values per row
//   - bool / other columns → DataPoint.Fields (preserved for expression evaluation)
func (r *AlertRule) FetchData() ([]*model.DataPoint, error) {
	// Re-parse the query to obtain a fresh time range.
	// queryParams is cached in Perpare() for compiled conditions, but Range must be
	// re-evaluated on every FetchData() so "now-5m" stays anchored to the current clock.
	freshQuery, err := r.rule.ToDataQuery()
	if err != nil {
		return nil, err
	}
	timerange := freshQuery.Range
	// If the rule uses the legacy array-only format, ToDataQuery returns a zero Range;
	// fall back to the cached range (which may itself be zero — callers should set a range).
	if timerange.Start == 0 && timerange.End == 0 {
		timerange = r.queryParams.Range
	}

	resp, err := r.querier.Query(r.queryParams.Queries, timerange)
	if err != nil {
		return nil, err
	}

	var rs []*model.DataPoint
	for refID, resultSet := range resp {
		if resultSet == nil || resultSet.Schema == nil {
			continue
		}
		cols := resultSet.Schema.Columns
		for _, row := range resultSet.Rows {
			if len(row) != len(cols) {
				continue
			}

			// Pass 1: collect dimension labels from string columns.
			labels := make(map[string]string)
			for i, col := range cols {
				if col.DataType == "string" {
					if v, ok := row[i].(string); ok {
						labels[col.Name] = v
					}
				}
			}

			// Pass 2: collect scalar numeric columns into a shared Values map.
			// All scalar columns from the same row are merged into one DataPoint so
			// that multi-field conditions like "cpu_usage > 80 && mem_usage > 90"
			// can be evaluated atomically.
			scalarValues := make(map[string]float64)
			scalarFields := make(map[string]any)
			for i, col := range cols {
				switch col.DataType {
				case "string", "time_series":
					// handled separately
					continue
				default:
					switch v := row[i].(type) {
					case float64:
						scalarValues[col.Name] = v
					case float32:
						scalarValues[col.Name] = float64(v)
					case int64:
						scalarValues[col.Name] = float64(v)
					case bool:
						scalarFields[col.Name] = v
					default:
						if row[i] != nil {
							scalarFields[col.Name] = row[i]
						}
					}
				}
			}

			// Pass 3: emit DataPoints.
			// time_series columns each produce one DataPoint per time point.
			// All scalar columns together produce exactly one DataPoint per row.
			hasScalars := len(scalarValues) > 0
			hasTimeSeries := false

			for i, col := range cols {
				if col.DataType != "time_series" {
					continue
				}
				ts, ok := row[i].(*ljson.TimeSeries)
				if !ok {
					continue
				}
				hasTimeSeries = true
				for j, v := range ts.Values {
					timestamp := ts.TimeRange.Start + int64(j)*ts.Interval
					dp := &model.DataPoint{
						RefID:     refID,
						Labels:    labels,
						Timestamp: timestamp,
						Value:     v,
						Values:    map[string]float64{col.Name: v},
					}
					// Merge same-row scalar values so cross-field expressions work on time points too.
					for k, sv := range scalarValues {
						dp.Values[k] = sv
					}
					if len(scalarFields) > 0 {
						dp.Fields = scalarFields
					}
					rs = append(rs, dp)
				}
			}

			if hasScalars && !hasTimeSeries {
				// No time_series column: emit one DataPoint for all scalar columns in this row.
				// Representative value: prefer the "value" column, else first numeric column.
				repValue := pickRepresentativeValue(scalarValues, cols)
				dp := &model.DataPoint{
					RefID:     refID,
					Labels:    labels,
					Values:    scalarValues,
					Timestamp: timerange.End, // scalar has no inherent timestamp; use query end
					Value:     repValue,
				}
				if len(scalarFields) > 0 {
					dp.Fields = scalarFields
				}
				rs = append(rs, dp)
			}
		}
	}
	r.ruleLogger.Info("data fetched",
		logger.String("rule", r.rule.UID),
		logger.Int32("queries", int32(len(r.queryParams.Queries))),
		logger.Int32("dataPoints", int32(len(rs))))
	return rs, nil
}

// pickRepresentativeValue selects the representative float value from a scalar row.
// Priority: column named "value" > first numeric column in schema order.
func pickRepresentativeValue(values map[string]float64, cols []ljson.ColumnMetadata) float64 {
	if v, ok := values["value"]; ok {
		return v
	}
	for _, col := range cols {
		if v, ok := values[col.Name]; ok {
			return v
		}
	}
	return 0
}

// Eval evaluates each condition against the fetched data points and returns the ones that matched.
func (r *AlertRule) Eval(ctx context.Context, dataPoints []*model.DataPoint) (matching []*model.DataPoint, err error) {
	for i := range dataPoints {
		dp := dataPoints[i]
		env := dp.ToMap()
		for idx := range r.conditions {
			output, err := expr.Run(r.conditions[idx], env)
			if err != nil {
				return nil, err
			}
			result, ok := output.(bool)
			if ok && result {
				dp.Severity = r.severities[idx] // record the severity of the matched condition
				matching = append(matching, dp)
				break // one matching condition is sufficient per data point
			}
		}
	}
	r.ruleLogger.Info("conditions evaluated",
		logger.String("rule", r.rule.UID),
		logger.Int32("total", int32(len(dataPoints))),
		logger.Int32("matched", int32(len(matching))))
	return matching, nil
}

// EvalAndTrack evaluates conditions, drives the state machine, and returns changed AlertEvents.
func (r *AlertRule) EvalAndTrack(ctx context.Context, dataPoints []*model.DataPoint, evalTime time.Time) ([]*model.AlertEvent, error) {
	matching, err := r.Eval(ctx, dataPoints)
	if err != nil {
		return nil, err
	}
	events := r.stateMgr.Eval(r.rule, matching, evalTime)
	for _, ev := range events {
		r.ruleLogger.Info("alert event",
			logger.String("rule", r.rule.UID),
			logger.Any("event", ev))
	}
	return events, nil
}

// ActiveAlerts returns the number of fingerprints currently in Pending or Firing state.
func (r *AlertRule) ActiveAlerts() int {
	return r.stateMgr.ActiveCount()
}
