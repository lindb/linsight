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

// FetchData queries the datasource and returns a flat list of Series data points.
// Each row in the Arrow ResultSet is processed as follows:
//   - "string" columns become Tags (group-by dimensions)
//   - "time_series" columns expand to one Series per data point (timestamp = start + i*interval)
//   - all other columns (aggregation scalars) become a single Series at the query start time
func (r *AlertRule) FetchData() (rs []*model.Series, err error) {
	resp, err := r.querier.Query(r.queryParams.Queries, r.queryParams.Range)
	if err != nil {
		return nil, err
	}

	for refID, resultSet := range resp {
		if resultSet == nil || resultSet.Schema == nil {
			continue
		}
		cols := resultSet.Schema.Columns
		for _, row := range resultSet.Rows {
			if len(row) != len(cols) {
				continue
			}
			// Collect tag values from string columns (group-by dimensions).
			tags := make(map[string]string)
			for i, col := range cols {
				if col.DataType == "string" {
					if v, ok := row[i].(string); ok {
						tags[col.Name] = v
					}
				}
			}
			// Extract metric values from non-tag columns.
			for i, col := range cols {
				if col.DataType == "string" {
					continue
				}
				switch col.DataType {
				case "time_series":
					ts, ok := row[i].(*ljson.TimeSeries)
					if !ok {
						continue
					}
					for j, v := range ts.Values {
						timestamp := ts.TimeRange.Start + int64(j)*ts.Interval
						rs = append(rs, &model.Series{
							RefID:     refID,
							Field:     col.Name,
							Tags:      tags,
							Timestamp: timestamp,
							Value:     v,
						})
					}
				default:
					// Aggregation (scalar) column — single data point at query start time.
					var val float64
					switch v := row[i].(type) {
					case float64:
						val = v
					case float32:
						val = float64(v)
					case int64:
						val = float64(v)
					default:
						continue
					}
					rs = append(rs, &model.Series{
						RefID:     refID,
						Field:     col.Name,
						Tags:      tags,
						Timestamp: r.queryParams.Range.Start,
						Value:     val,
					})
				}
			}
		}
	}
	r.ruleLogger.Info("data fetched",
		logger.String("rule", r.rule.UID),
		logger.Int32("queries", int32(len(r.queryParams.Queries))),
		logger.Int32("series", int32(len(rs))))
	return
}

// Eval evaluates each condition against the fetched series and returns the series that matched.
func (r *AlertRule) Eval(ctx context.Context, seriesList []*model.Series) (matchingSeries []*model.Series, err error) {
	for i := range seriesList {
		series := seriesList[i]
		env := series.ToMap()
		for idx := range r.conditions {
			output, err := expr.Run(r.conditions[idx], env)
			if err != nil {
				return nil, err
			}
			result, ok := output.(bool)
			if ok && result {
				series.Severity = r.severities[idx] // record the severity of the matched condition
				matchingSeries = append(matchingSeries, series)
				break // one matching condition is sufficient per series
			}
		}
	}
	r.ruleLogger.Info("conditions evaluated",
		logger.String("rule", r.rule.UID),
		logger.Int32("total", int32(len(seriesList))),
		logger.Int32("matched", int32(len(matchingSeries))))
	return matchingSeries, nil
}

// EvalAndTrack evaluates conditions, drives the state machine, and returns changed AlertEvents.
func (r *AlertRule) EvalAndTrack(ctx context.Context, seriesList []*model.Series, evalTime time.Time) ([]*model.AlertEvent, error) {
	matchingSeries, err := r.Eval(ctx, seriesList)
	if err != nil {
		return nil, err
	}
	events := r.stateMgr.Eval(r.rule, matchingSeries, evalTime)
	return events, nil
}

// ActiveAlerts returns the number of fingerprints currently in Pending or Firing state.
func (r *AlertRule) ActiveAlerts() int {
	return r.stateMgr.ActiveCount()
}
