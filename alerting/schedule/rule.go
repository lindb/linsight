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
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// AlertRule wraps a model.AlertRule with compiled conditions and cached query parameters.
type AlertRule struct {
	conditions   []*vm.Program
	queryParams  *model.QueryRequest
	rule         *model.AlertRule
	querier      Querier
	stateMgr     RuleStateManager

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

	r.conditions = make([]*vm.Program, len(conditions))
	r.queryParams = dataQuery

	for idx := range conditions {
		condition := conditions[idx]
		var str string
		if err := encoding.JSONUnmarshal(condition.Expr, &str); err != nil {
			return err
		}
		r.ruleLogger.Debug("compile condition", logger.String("rule", rule.UID), logger.String("expr", str))
		program, err := expr.Compile(str)
		if err != nil {
			return err
		}
		r.conditions[idx] = program
	}

	r.currentVersion = rule.Version
	return nil
}

// FetchData queries the datasource and returns one Series per field per metric series.
func (r *AlertRule) FetchData() (rs []*model.Series, err error) {
	resp, err := r.querier.Query(r.queryParams.Queries, r.queryParams.Range)
	if err != nil {
		return nil, err
	}

	for refID, resultSet := range resp {
		for _, s := range resultSet.Series {
			for fn, f := range s.Fields {
				ts, val := getValue(f)
				rs = append(rs, &model.Series{
					RefID:      refID,
					MetricName: resultSet.MetricName,
					Field:      fn,
					Tags:       s.Tags,
					Timestamp:  ts,
					Value:      val,
				})
			}
		}
	}
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
				matchingSeries = append(matchingSeries, series)
				break // one matching condition is sufficient per series
			}
		}
	}
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
