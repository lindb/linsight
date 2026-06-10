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
	"time"

	"github.com/lindb/linsight/model"
)

// DryRunner executes the full alert rule evaluation pipeline once without any side effects.
// It reuses the same Perpare → FetchData → EvalAndTrack chain as the live scheduler,
// but the in-memory RuleStateManager used by EvalAndTrack never writes to the database,
// so the result is purely diagnostic.
type DryRunner interface {
	Run(ctx context.Context, rule *model.AlertRule) (*model.AlertRuleDryRunResult, error)
}

type dryRunner struct {
	deps *Deps
}

// NewDryRunner creates a DryRunner backed by the given scheduling dependencies.
// Only DatasourceSrv and DatasourceMgr are required; AlertMgr and NotificationSrv
// can be nil because dry-run never delivers notifications.
func NewDryRunner(deps *Deps) DryRunner {
	return &dryRunner{deps: deps}
}

// Run executes the evaluation pipeline for the given rule and returns the diagnostic result.
// Errors at each stage are captured inside the result (Stage + Error fields) rather than
// returned as a Go error so the caller always receives a structured response.
func (d *dryRunner) Run(ctx context.Context, rule *model.AlertRule) (*model.AlertRuleDryRunResult, error) {
	alertRule := NewAlertRule(rule, d.deps)

	// Stage 1: compile conditions and decode query parameters.
	if err := alertRule.Perpare(); err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "prepare",
			Error: err.Error(),
		}, nil
	}

	// Stage 2: fetch raw series from the configured datasource.
	seriesList, err := alertRule.FetchData()
	if err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "fetch",
			Error: err.Error(),
		}, nil
	}

	// Stage 3: evaluate conditions and run the in-memory state machine.
	// EvalAndTrack uses a fresh RuleStateManager (no DB), so no events are persisted.
	events, err := alertRule.EvalAndTrack(ctx, seriesList, time.Now())
	if err != nil {
		return &model.AlertRuleDryRunResult{
			Stage: "eval",
			Error: err.Error(),
		}, nil
	}

	return &model.AlertRuleDryRunResult{
		Series: seriesList,
		Events: events,
	}, nil
}
