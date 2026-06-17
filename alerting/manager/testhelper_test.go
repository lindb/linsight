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

package manager

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lindb/common/pkg/logger"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
)

// ─── Test stage stubs ────────────────────────────────────────────────────────

// passThroughStage passes all events through and records how many times it was called.
type passThroughStage struct{ called int }

func (s *passThroughStage) Run(_ *stageCtx, events []*model.AlertEvent) []*model.AlertEvent {
	s.called++
	return events
}

// dropAllStage drops all events and records how many times it was called.
type dropAllStage struct{ called int }

func (s *dropAllStage) Run(_ *stageCtx, events []*model.AlertEvent) []*model.AlertEvent {
	s.called++
	return nil
}

// ─── Event builders ──────────────────────────────────────────────────────────

// mustMatchersJSON serializes a slice of LabelMatcher to datatypes.JSON.
func mustMatchersJSON(t *testing.T, ms []model.LabelMatcher) datatypes.JSON {
	t.Helper()
	b, err := json.Marshal(ms)
	require.NoError(t, err)
	return datatypes.JSON(b)
}

// makeEvent builds an AlertEvent whose Labels field is JSON-encoded from the given map.
func makeEvent(labels map[string]string) *model.AlertEvent {
	b, _ := json.Marshal(labels)
	return &model.AlertEvent{Labels: datatypes.JSON(b)}
}

// makeStateEvent builds an AlertEvent with fingerprint, state, and labels.
func makeStateEvent(fingerprint string, state model.AlertEventState, labels map[string]string) *model.AlertEvent {
	b, _ := json.Marshal(labels)
	return &model.AlertEvent{
		Fingerprint: fingerprint,
		State:       state,
		Labels:      datatypes.JSON(b),
	}
}

// ─── Silence / inhibit builders ──────────────────────────────────────────────

// makeSilence constructs an AlertSilence with the given matchers and [startsAt, endsAt) window.
func makeSilence(matchers []model.LabelMatcher, startsAt, endsAt int64) *model.AlertSilence {
	b, _ := json.Marshal(matchers)
	return &model.AlertSilence{
		Matchers: datatypes.JSON(b),
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}
}

// makeInhibitRule constructs an AlertInhibitRule from source matchers, target matchers, and equal labels.
func makeInhibitRule(src, tgt []model.LabelMatcher, equal []string) *model.AlertInhibitRule {
	srcB, _ := json.Marshal(src)
	tgtB, _ := json.Marshal(tgt)
	eqB, _ := json.Marshal(equal)
	return &model.AlertInhibitRule{
		SourceMatchers: datatypes.JSON(srcB),
		TargetMatchers: datatypes.JSON(tgtB),
		EqualLabels:    datatypes.JSON(eqB),
	}
}

// ─── Route tree builder ──────────────────────────────────────────────────────

// buildTestTree constructs the following route tree without touching the DB or service layer:
//
//	root  (ContactPoint="default-email", GroupBy=["alertname"], GroupWait=30s,
//	        GroupInterval=5m, RepeatInterval=4h)
//	├── db-route  (service="database", continue=false, ContactPoint="dba-default-slack")
//	│   ├── db-prod  (env="production", ContactPoint="dba-prod-phone")
//	│   └── db-test  (env="testing",    ContactPoint="dba-test-mutilated")
//	└── critical-route (severity="critical", ContactPoint="global-oncall-pager")
func buildTestTree(t *testing.T) *model.AlertRouteNode {
	t.Helper()

	root := &model.AlertRouteNode{
		AlertRoute: model.AlertRoute{
			UID:             "root",
			ParentUID:       "",
			ContactPointUID: "default-email",
			GroupBy:         datatypes.JSON(`["alertname"]`),
			GroupWait:       30 * time.Second,
			GroupInterval:   5 * time.Minute,
			RepeatInterval:  4 * time.Hour,
		},
	}

	dbRoute := &model.AlertRouteNode{
		AlertRoute: model.AlertRoute{
			UID:             "db-route",
			ParentUID:       "root",
			ContactPointUID: "dba-default-slack",
			Continue:        false,
			Matchers: mustMatchersJSON(t, []model.LabelMatcher{
				{Name: "service", Value: "database", Type: model.MatchEqual},
			}),
		},
	}

	dbProd := &model.AlertRouteNode{
		AlertRoute: model.AlertRoute{
			UID:             "db-prod",
			ParentUID:       "db-route",
			ContactPointUID: "dba-prod-phone",
			Matchers: mustMatchersJSON(t, []model.LabelMatcher{
				{Name: "env", Value: "production", Type: model.MatchEqual},
			}),
		},
	}

	dbTest := &model.AlertRouteNode{
		AlertRoute: model.AlertRoute{
			UID:             "db-test",
			ParentUID:       "db-route",
			ContactPointUID: "dba-test-mutilated",
			Matchers: mustMatchersJSON(t, []model.LabelMatcher{
				{Name: "env", Value: "testing", Type: model.MatchEqual},
			}),
		},
	}

	criticalRoute := &model.AlertRouteNode{
		AlertRoute: model.AlertRoute{
			UID:             "critical-route",
			ParentUID:       "root",
			ContactPointUID: "global-oncall-pager",
			Matchers: mustMatchersJSON(t, []model.LabelMatcher{
				{Name: "severity", Value: "critical", Type: model.MatchEqual},
			}),
		},
	}

	dbRoute.Routes = []*model.AlertRouteNode{dbProd, dbTest}
	root.Routes = []*model.AlertRouteNode{dbRoute, criticalRoute}
	return root
}

// contactPoints extracts ContactPointUID from a slice of routes for easy assertion.
func contactPoints(routes []*model.AlertRoute) []string {
	out := make([]string, 0, len(routes))
	for _, r := range routes {
		out = append(out, r.ContactPointUID)
	}
	return out
}

// testLogger returns a logger suitable for use in stage tests.
func testLogger() logger.Logger {
	return logger.GetLogger("Test", "Manager")
}
