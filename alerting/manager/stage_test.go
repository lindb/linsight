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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
)

// ─── SilenceStage tests ──────────────────────────────────────────────────────

func TestSilenceStage_ActiveSilence_FiltersEvent(t *testing.T) {
	now := int64(1000)
	silence := makeSilence([]model.LabelMatcher{
		{Name: "alertname", Value: "high-cpu", Type: model.MatchEqual},
	}, 500, 2000) // active: startsAt=500 <= now=1000 < endsAt=2000

	stage := &SilenceStage{silences: []*model.AlertSilence{silence}, now: now, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{makeEvent(map[string]string{"alertname": "high-cpu"})})
	assert.Empty(t, result, "matched event should be silenced")
}

func TestSilenceStage_ExpiredSilence_PassesEvent(t *testing.T) {
	now := int64(3000)
	silence := makeSilence([]model.LabelMatcher{
		{Name: "alertname", Value: "high-cpu", Type: model.MatchEqual},
	}, 500, 2000) // expired: endsAt=2000 <= now=3000

	stage := &SilenceStage{silences: []*model.AlertSilence{silence}, now: now, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{makeEvent(map[string]string{"alertname": "high-cpu"})})
	assert.Len(t, result, 1, "expired silence must not filter events")
}

func TestSilenceStage_FutureSilence_PassesEvent(t *testing.T) {
	now := int64(100)
	silence := makeSilence([]model.LabelMatcher{
		{Name: "alertname", Value: "high-cpu", Type: model.MatchEqual},
	}, 500, 2000) // not yet started: startsAt=500 > now=100

	stage := &SilenceStage{silences: []*model.AlertSilence{silence}, now: now, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{makeEvent(map[string]string{"alertname": "high-cpu"})})
	assert.Len(t, result, 1, "future silence must not filter events")
}

func TestSilenceStage_MatcherMismatch_PassesEvent(t *testing.T) {
	now := int64(1000)
	silence := makeSilence([]model.LabelMatcher{
		{Name: "alertname", Value: "high-cpu", Type: model.MatchEqual},
	}, 500, 2000)

	stage := &SilenceStage{silences: []*model.AlertSilence{silence}, now: now, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{makeEvent(map[string]string{"alertname": "disk-full"})})
	assert.Len(t, result, 1, "non-matching silence must not filter events")
}

func TestSilenceStage_MultipleEvents_OnlyMatchedFiltered(t *testing.T) {
	now := int64(1000)
	silence := makeSilence([]model.LabelMatcher{
		{Name: "env", Value: "testing", Type: model.MatchEqual},
	}, 500, 2000)

	stage := &SilenceStage{silences: []*model.AlertSilence{silence}, now: now, logger: testLogger()}
	evTesting := makeEvent(map[string]string{"env": "testing"})
	evProd := makeEvent(map[string]string{"env": "production"})

	result := stage.Run(nil, []*model.AlertEvent{evTesting, evProd})
	require.Len(t, result, 1, "only the silenced event should be removed")
	assert.Equal(t, evProd, result[0])
}

// ─── InhibitStage tests ──────────────────────────────────────────────────────

func TestInhibitStage_SameBatch_SourceFiring_Inhibited(t *testing.T) {
	rule := makeInhibitRule(
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]model.LabelMatcher{{Name: "severity", Value: "warning", Type: model.MatchEqual}},
		[]string{"alertname"},
	)
	srcEv := makeStateEvent("fp-src", model.AlertEventFiring, map[string]string{"severity": "critical", "alertname": "disk"})
	tgtEv := makeStateEvent("fp-tgt", model.AlertEventFiring, map[string]string{"severity": "warning", "alertname": "disk"})

	stage := &InhibitStage{inhibits: []*model.AlertInhibitRule{rule}, allFiring: []*model.AlertEvent{srcEv, tgtEv}, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{tgtEv})
	assert.Empty(t, result, "target should be inhibited by same-batch source")
}

// TestInhibitStage_CrossBatch_SourceFiring_Inhibited verifies the P0 fix:
// a source alert that fired in a previous batch (present in allFiring but not in the current
// batch) must still inhibit the target.
func TestInhibitStage_CrossBatch_SourceFiring_Inhibited(t *testing.T) {
	rule := makeInhibitRule(
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]model.LabelMatcher{{Name: "severity", Value: "warning", Type: model.MatchEqual}},
		[]string{"alertname"},
	)
	prevBatchSrc := makeStateEvent("fp-src", model.AlertEventFiring, map[string]string{"severity": "critical", "alertname": "disk"})
	tgtEv := makeStateEvent("fp-tgt", model.AlertEventFiring, map[string]string{"severity": "warning", "alertname": "disk"})

	stage := &InhibitStage{
		inhibits:  []*model.AlertInhibitRule{rule},
		allFiring: []*model.AlertEvent{prevBatchSrc, tgtEv}, // merged DB firing + current batch
		logger:    testLogger(),
	}
	result := stage.Run(nil, []*model.AlertEvent{tgtEv})
	assert.Empty(t, result, "target should be inhibited even when source is from a previous batch")
}

func TestInhibitStage_EqualLabelMismatch_NotInhibited(t *testing.T) {
	rule := makeInhibitRule(
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]model.LabelMatcher{{Name: "severity", Value: "warning", Type: model.MatchEqual}},
		[]string{"alertname"},
	)
	// Source and target have different alertname values → equal-label constraint not satisfied.
	srcEv := makeStateEvent("fp-src", model.AlertEventFiring, map[string]string{"severity": "critical", "alertname": "cpu"})
	tgtEv := makeStateEvent("fp-tgt", model.AlertEventFiring, map[string]string{"severity": "warning", "alertname": "disk"})

	stage := &InhibitStage{inhibits: []*model.AlertInhibitRule{rule}, allFiring: []*model.AlertEvent{srcEv, tgtEv}, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{tgtEv})
	assert.Len(t, result, 1, "equal-label mismatch: target must not be inhibited")
}

func TestInhibitStage_TargetNotMatched_NotInhibited(t *testing.T) {
	rule := makeInhibitRule(
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]model.LabelMatcher{{Name: "severity", Value: "warning", Type: model.MatchEqual}},
		[]string{"alertname"},
	)
	srcEv := makeStateEvent("fp-src", model.AlertEventFiring, map[string]string{"severity": "critical", "alertname": "disk"})
	// Target has severity=info, which does NOT match the target matcher (severity=warning).
	tgtEv := makeStateEvent("fp-tgt", model.AlertEventFiring, map[string]string{"severity": "info", "alertname": "disk"})

	stage := &InhibitStage{inhibits: []*model.AlertInhibitRule{rule}, allFiring: []*model.AlertEvent{srcEv, tgtEv}, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{tgtEv})
	assert.Len(t, result, 1, "event that does not match target matchers must not be inhibited")
}

func TestInhibitStage_SelfInhibit_NotInhibited(t *testing.T) {
	// An event must not inhibit itself (same-fingerprint guard).
	rule := makeInhibitRule(
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]model.LabelMatcher{{Name: "severity", Value: "critical", Type: model.MatchEqual}},
		[]string{"alertname"},
	)
	ev := makeStateEvent("fp-same", model.AlertEventFiring, map[string]string{"severity": "critical", "alertname": "disk"})

	stage := &InhibitStage{inhibits: []*model.AlertInhibitRule{rule}, allFiring: []*model.AlertEvent{ev}, logger: testLogger()}
	result := stage.Run(nil, []*model.AlertEvent{ev})
	assert.Len(t, result, 1, "an event must not inhibit itself")
}

// ─── Pipeline tests ──────────────────────────────────────────────────────────

func TestPipeline_EmptyEvents_StagesNotRun(t *testing.T) {
	stage1 := &passThroughStage{}
	pipeline := Pipeline{stage1}
	pipeline.Run(&stageCtx{}, []*model.AlertEvent{})
	assert.Equal(t, 0, stage1.called, "stages must not run when the initial event set is empty")
}

func TestPipeline_AllDropped_SubsequentStageSkipped(t *testing.T) {
	dropAll := &dropAllStage{}
	passThru := &passThroughStage{}
	pipeline := Pipeline{dropAll, passThru}

	pipeline.Run(&stageCtx{}, []*model.AlertEvent{makeEvent(nil)})

	assert.Equal(t, 1, dropAll.called, "first stage must run")
	assert.Equal(t, 0, passThru.called, "stage after drop-all must be skipped")
}

func TestPipeline_MultipleStages_AllRun(t *testing.T) {
	s1 := &passThroughStage{}
	s2 := &passThroughStage{}
	pipeline := Pipeline{s1, s2}

	pipeline.Run(&stageCtx{}, []*model.AlertEvent{makeEvent(nil)})

	assert.Equal(t, 1, s1.called)
	assert.Equal(t, 1, s2.called)
}

// ─── mergeFiring tests ───────────────────────────────────────────────────────

func TestMergeFiring_OnlyFiringFromBatch(t *testing.T) {
	batch := []*model.AlertEvent{
		makeStateEvent("fp1", model.AlertEventFiring, nil),
		makeStateEvent("fp2", model.AlertEventPending, nil),  // excluded: not firing
		makeStateEvent("fp3", model.AlertEventResolved, nil), // excluded: not firing
	}
	result := mergeFiring(nil, batch)
	require.Len(t, result, 1)
	assert.Equal(t, "fp1", result[0].Fingerprint)
}

func TestMergeFiring_Dedup_BatchTakesPrecedence(t *testing.T) {
	// Same fingerprint in both DB and batch; batch copy must win.
	dbEv := makeStateEvent("fp1", model.AlertEventFiring, map[string]string{"src": "db"})
	batchEv := makeStateEvent("fp1", model.AlertEventFiring, map[string]string{"src": "batch"})

	result := mergeFiring([]*model.AlertEvent{dbEv}, []*model.AlertEvent{batchEv})
	require.Len(t, result, 1, "duplicate fingerprint must be deduplicated")
	assert.Equal(t, "batch", parseEventLabels(result[0])["src"])
}

func TestMergeFiring_DBOnlyFiring_Included(t *testing.T) {
	dbEv := makeStateEvent("fp-db", model.AlertEventFiring, nil)
	result := mergeFiring([]*model.AlertEvent{dbEv}, nil)
	require.Len(t, result, 1)
	assert.Equal(t, "fp-db", result[0].Fingerprint)
}

func TestMergeFiring_BothSources_UnionResult(t *testing.T) {
	dbEv := makeStateEvent("fp-db", model.AlertEventFiring, nil)
	batchEv := makeStateEvent("fp-batch", model.AlertEventFiring, nil)
	result := mergeFiring([]*model.AlertEvent{dbEv}, []*model.AlertEvent{batchEv})
	assert.Len(t, result, 2)
}

// ─── buildGroupKey tests ─────────────────────────────────────────────────────

func TestBuildGroupKey_WithGroupBy(t *testing.T) {
	route := &model.AlertRoute{
		UID:     "route-1",
		GroupBy: datatypes.JSON(`["alertname","env"]`),
	}
	key := buildGroupKey(42, route, makeEvent(map[string]string{"alertname": "high-cpu", "env": "prod"}))

	assert.Contains(t, key, "alertname=high-cpu;")
	assert.Contains(t, key, "env=prod;")
	assert.True(t, strings.HasSuffix(key, ":route-1"), "key must end with :<routeUID>")
}

func TestBuildGroupKey_WithoutGroupBy_RouteUIDOnly(t *testing.T) {
	key := buildGroupKey(1, &model.AlertRoute{UID: "route-2"}, makeEvent(nil))
	assert.Equal(t, ":route-2", key)
}

func TestBuildGroupKey_DifferentRoutes_DifferentKeys(t *testing.T) {
	ev := makeEvent(map[string]string{"alertname": "disk"})
	r1 := &model.AlertRoute{UID: "r1", GroupBy: datatypes.JSON(`["alertname"]`)}
	r2 := &model.AlertRoute{UID: "r2", GroupBy: datatypes.JSON(`["alertname"]`)}
	assert.NotEqual(t, buildGroupKey(1, r1, ev), buildGroupKey(1, r2, ev))
}
