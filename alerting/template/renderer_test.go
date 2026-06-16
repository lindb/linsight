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

package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lindb/linsight/model"
)

// buildFiringAlert creates a minimal TemplateAlert with the given rule title, instance label, and value.
func buildFiringAlert(ruleTitle, instance string, value float64) TemplateAlert {
	return TemplateAlert{
		RuleTitle: ruleTitle,
		State:     model.AlertEventFiring,
		Status:    "firing",
		Labels:    map[string]string{"instance": instance},
		Value:     value,
	}
}

// ─── Renderer.Render ─────────────────────────────────────────────────────────

func TestRenderer_SimpleVariableInterpolation(t *testing.T) {
	// Verifies that basic variable interpolation and the "upper" template function work.
	r := NewRenderer()
	data := &NotificationData{
		Status:       "firing",
		ReceiverName: "slack-ops",
		CommonLabels: map[string]string{
			"alertname": "HighCPU",
			"instance":  "web-01",
		},
	}
	result, err := r.Render(`[{{ .Status | upper }}] {{ .CommonLabels.alertname }}`, data)
	require.NoError(t, err)
	assert.Equal(t, "[FIRING] HighCPU", result)
}

func TestRenderer_NamedDefineBlock(t *testing.T) {
	// Verifies that {{define}} and {{template}} work within a single template string.
	r := NewRenderer()
	data := &NotificationData{Status: "resolved"}
	tmplStr := `{{define "status"}}Status: {{ .Status | title }}{{end}}{{ template "status" . }}`
	result, err := r.Render(tmplStr, data)
	require.NoError(t, err)
	assert.Equal(t, "Status: Resolved", result)
}

func TestRenderer_RangeFiringAlerts(t *testing.T) {
	// Verifies iterating over FiringAlerts with direct dot-notation label access.
	r := NewRenderer()
	data := &NotificationData{
		FiringAlerts: []TemplateAlert{
			buildFiringAlert("HighCPU", "web-01", 92.5),
			buildFiringAlert("LowDisk", "db-01", 5.0),
		},
	}
	tmplStr := `{{ range .FiringAlerts }}[{{ .RuleTitle }}] on {{ .Labels.instance }}
{{ end }}`
	result, err := r.Render(tmplStr, data)
	require.NoError(t, err)
	assert.Contains(t, result, "[HighCPU] on web-01")
	assert.Contains(t, result, "[LowDisk] on db-01")
}

func TestRenderer_DefaultTitle(t *testing.T) {
	// Verifies the built-in title template renders alertname and instance.
	r := NewRenderer()
	data := &NotificationData{
		Status: "firing",
		CommonLabels: map[string]string{
			"alertname": "DiskFull",
			"instance":  "storage-01",
		},
	}
	result, err := r.RenderDefaultTitle(data)
	require.NoError(t, err)
	assert.Contains(t, result, "DiskFull")
	assert.Contains(t, result, "storage-01")
}

func TestRenderer_DefaultBody_FiringAndResolved(t *testing.T) {
	// Verifies the built-in body template shows firing and resolved sections.
	r := NewRenderer()
	data := &NotificationData{
		Status:       "firing",
		ReceiverName: "pagerduty",
		FiringAlerts: []TemplateAlert{
			buildFiringAlert("HighCPU", "web-01", 99),
		},
		ResolvedAlerts: []TemplateAlert{
			{RuleTitle: "OldAlert", State: model.AlertEventResolved, Status: "resolved"},
		},
	}
	result, err := r.RenderDefaultBody(data)
	require.NoError(t, err)
	assert.Contains(t, result, "HighCPU")
	assert.Contains(t, result, "[RESOLVED] OldAlert")
}

func TestRenderer_InvalidTemplateSyntax(t *testing.T) {
	// Verifies that a syntactically broken template returns a parse error.
	r := NewRenderer()
	_, err := r.Render(`{{ .Foo`, &NotificationData{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse template")
}

// ─── Template Functions ───────────────────────────────────────────────────────

func TestTemplateFuncs_Join(t *testing.T) {
	// Accessing a non-existent map key in Go template returns reflect.Value{} (zero Value),
	// which the engine passes to join as a nil []string — so Execute succeeds and returns "".
	// This test verifies no panic occurs and the result is an empty string.
	r := NewRenderer()
	result, err := r.Render(`{{ join ", " .GroupLabels.keys }}`, &NotificationData{
		GroupLabels: map[string]string{"env": "prod"},
	})
	require.NoError(t, err)
	// map has no "keys" key; join receives nil slice, result is ""
	assert.Equal(t, "", result)
}

func TestTemplateFuncs_Match(t *testing.T) {
	r := NewRenderer()
	result, err := r.Render(`{{ if match "crit.*" "critical" }}yes{{ else }}no{{ end }}`, &NotificationData{})
	require.NoError(t, err)
	assert.Equal(t, "yes", result)
}

func TestTemplateFuncs_ReReplaceAll(t *testing.T) {
	r := NewRenderer()
	data := &NotificationData{
		CommonLabels: map[string]string{"instance": "web-123"},
	}
	result, err := r.Render(`{{ reReplaceAll "web-\\d+" "web-XX" .CommonLabels.instance }}`, data)
	require.NoError(t, err)
	assert.Equal(t, "web-XX", result)
}

func TestTemplateFuncs_ReReplaceAll_InvalidRegex(t *testing.T) {
	// When the pattern is not a valid regex, reReplaceAll must return the original
	// string unchanged instead of panicking (MustCompile would panic).
	r := NewRenderer()
	result, err := r.Render(`{{ reReplaceAll "[invalid" "x" .ReceiverName }}`, &NotificationData{ReceiverName: "original"})
	require.NoError(t, err)
	assert.Equal(t, "original", result, "invalid regex must return original string, not panic")
}

func TestTemplateFuncs_HumanizeDuration(t *testing.T) {
	r := NewRenderer()
	cases := []struct {
		ms       int64
		expected string
	}{
		{3661000, "1h1m1s"},
		{65000, "1m5s"},
		{5000, "5s"},
		{0, "0s"},
	}
	for _, tt := range cases {
		result, err := r.Render(
			`{{ humanizeDuration .Value }}`,
			&NotificationData{FiringAlerts: []TemplateAlert{{Value: float64(tt.ms)}}},
		)
		// humanizeDuration takes int64; .Value is float64 — type mismatch causes execute error.
		// The canonical test below uses a literal integer directly in the template.
		_ = result
		_ = err
	}

	// Canonical correctness checks using inline integer literals.
	cases2 := []struct{ tmpl, want string }{
		{`{{ humanizeDuration 3661000 }}`, "1h1m1s"},
		{`{{ humanizeDuration 65000 }}`, "1m5s"},
		{`{{ humanizeDuration 5000 }}`, "5s"},
		{`{{ humanizeDuration 0 }}`, "0s"},
	}
	for _, tc := range cases2 {
		result, err := r.Render(tc.tmpl, &NotificationData{})
		require.NoError(t, err)
		assert.Equal(t, tc.want, result, "template: %s", tc.tmpl)
	}
}

func TestTemplateFuncs_TrimSpace(t *testing.T) {
	r := NewRenderer()
	data := &NotificationData{ReceiverName: "  slack  "}
	result, err := r.Render(`{{ .ReceiverName | trimSpace }}`, data)
	require.NoError(t, err)
	assert.Equal(t, "slack", result)
}

func TestTemplateFuncs_Title(t *testing.T) {
	r := NewRenderer()
	data := &NotificationData{Status: "firing alert"}
	result, err := r.Render(`{{ .Status | title }}`, data)
	require.NoError(t, err)
	assert.Equal(t, "Firing Alert", result)
}

// ─── alertEventToTemplateAlert ────────────────────────────────────────────────

func TestAlertEventToTemplateAlert_States(t *testing.T) {
	// Verifies Status string derivation for each AlertEventState.
	cases := []struct {
		state          model.AlertEventState
		expectedStatus string
	}{
		{model.AlertEventFiring, "firing"},
		{model.AlertEventPending, "firing"},
		{model.AlertEventResolved, "resolved"},
		{model.AlertEventNoData, "resolved"},  // NoData is treated as resolved in alertEventToTemplateAlert
		{model.AlertEventError, "resolved"},    // Error is treated as resolved in alertEventToTemplateAlert
	}
	for _, tc := range cases {
		ev := &model.AlertEvent{State: tc.state}
		ta := alertEventToTemplateAlert(ev)
		assert.Equal(t, tc.expectedStatus, ta.Status, "state=%d", tc.state)
		assert.Equal(t, tc.state, ta.State)
	}
}

// ─── BuildNotificationData ────────────────────────────────────────────────────

func TestBuildNotificationData_NoDataAndError(t *testing.T) {
	// Verifies that NoData and Error states are classified as FiringAlerts,
	// not silently dropped.
	events := []*model.AlertEvent{
		{RuleTitle: "NoDataAlert", State: model.AlertEventNoData},
		{RuleTitle: "ErrorAlert", State: model.AlertEventError},
		{RuleTitle: "ResolvedAlert", State: model.AlertEventResolved},
	}
	data := BuildNotificationData(events, "test", "firing", "", nil)
	require.Len(t, data.FiringAlerts, 2, "NoData and Error must be in FiringAlerts")
	require.Len(t, data.ResolvedAlerts, 1)
	assert.Equal(t, "NoDataAlert", data.FiringAlerts[0].RuleTitle)
	assert.Equal(t, "ErrorAlert", data.FiringAlerts[1].RuleTitle)
}

func TestBuildNotificationData_CommonLabelsNeverNil(t *testing.T) {
	// CommonLabels/CommonAnnotations must be non-nil even when there are no alerts,
	// so templates can safely range over them without checking for nil.
	data := BuildNotificationData(nil, "", "", "", nil)
	assert.NotNil(t, data.CommonLabels)
	assert.NotNil(t, data.CommonAnnotations)
}

// ─── BuildNotificationDataFromPayload ─────────────────────────────────────────

func TestBuildNotificationDataFromPayload_AutoDeriveFiringAndResolved(t *testing.T) {
	// Verifies that FiringAlerts/ResolvedAlerts are auto-derived from Alerts when absent.
	payload := `{
		"alerts": [
			{"ruleTitle":"HighCPU","state":2,"labels":{"instance":"web-01"},"value":92.5},
			{"ruleTitle":"OldAlert","state":3,"labels":{"instance":"db-01"},"value":0}
		],
		"receiverName": "pagerduty",
		"status": "firing"
	}`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	require.Len(t, data.FiringAlerts, 1)
	require.Len(t, data.ResolvedAlerts, 1)
	assert.Equal(t, "HighCPU", data.FiringAlerts[0].RuleTitle)
	assert.Equal(t, "OldAlert", data.ResolvedAlerts[0].RuleTitle)
	assert.Equal(t, "pagerduty", data.ReceiverName)
	assert.Equal(t, "firing", data.Status)
}

func TestBuildNotificationDataFromPayload_AutoDeriveStatus(t *testing.T) {
	// When "status" is absent from the object payload but alerts are present,
	// status must be derived from the alert states.
	payloadFiring := `{"alerts":[{"ruleTitle":"A","state":2}]}`
	data, err := BuildNotificationDataFromPayload(payloadFiring)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)

	payloadResolved := `{"alerts":[{"ruleTitle":"B","state":3}]}`
	data, err = BuildNotificationDataFromPayload(payloadResolved)
	require.NoError(t, err)
	assert.Equal(t, "resolved", data.Status)
}

func TestBuildNotificationDataFromPayload_ExplicitSubLists(t *testing.T) {
	// When firingAlerts/resolvedAlerts are explicitly set, they should NOT be re-derived.
	payload := `{
		"firingAlerts": [{"ruleTitle":"ExplicitFiring","state":2}],
		"resolvedAlerts": [],
		"status": "firing"
	}`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Len(t, data.FiringAlerts, 1)
	assert.Equal(t, "ExplicitFiring", data.FiringAlerts[0].RuleTitle)
}

func TestBuildNotificationDataFromPayload_EmptyPayload(t *testing.T) {
	// Empty string should return an empty NotificationData without error.
	data, err := BuildNotificationDataFromPayload("")
	require.NoError(t, err)
	assert.NotNil(t, data)
	assert.Empty(t, data.Alerts)
}

func TestBuildNotificationDataFromPayload_InvalidJSON(t *testing.T) {
	_, err := BuildNotificationDataFromPayload(`{invalid`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse notification payload")
}

func TestBuildNotificationDataFromPayload_AutoDeriveCommonLabels(t *testing.T) {
	// CommonLabels should be derived as the intersection of labels across all alerts.
	payload := `{
		"alerts": [
			{"ruleTitle":"A","state":2,"labels":{"env":"prod","region":"us"}},
			{"ruleTitle":"B","state":2,"labels":{"env":"prod","region":"eu"}}
		],
		"status": "firing"
	}`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	// "env=prod" is common; "region" differs so should be excluded
	assert.Equal(t, "prod", data.CommonLabels["env"])
	assert.NotContains(t, data.CommonLabels, "region")
}

func TestBuildNotificationDataFromPayload_NullElementsInArray(t *testing.T) {
	// JSON arrays may contain explicit null values; they must be silently skipped
	// without causing a nil-pointer panic.
	payload := `[{"ruleTitle":"A","state":2}, null, {"ruleTitle":"B","state":3}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Len(t, data.Alerts, 2, "null elements must be skipped")
}

func TestBuildNotificationDataFromPayload_ArrayFormat(t *testing.T) {
	// Verifies that a raw []TemplateAlert array payload (LinSight format) is accepted
	// and status/lists are auto-derived from the "state" int field.
	payload := `[{"ruleTitle":"HighCPU","state":2,"labels":{"alertname":"HighCPU","instance":"web-01"},"value":95.5}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)
	require.Len(t, data.FiringAlerts, 1)
	assert.Equal(t, "HighCPU", data.FiringAlerts[0].RuleTitle)
	assert.Equal(t, "HighCPU", data.CommonLabels["alertname"])
}

func TestBuildNotificationDataFromPayload_ArrayFormatAllResolved(t *testing.T) {
	// Verifies that an all-resolved array payload derives status="resolved".
	payload := `[{"ruleTitle":"OldAlert","state":3}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "resolved", data.Status)
	assert.Empty(t, data.FiringAlerts)
	require.Len(t, data.ResolvedAlerts, 1)
	assert.Equal(t, "OldAlert", data.ResolvedAlerts[0].RuleTitle)
}

func TestBuildNotificationDataFromPayload_ArrayFormat_ISO8601(t *testing.T) {
	// Verifies that an array payload with ISO 8601 timestamp strings is parsed correctly.
	// Alertmanager sends "0001-01-01T00:00:00Z" for endsAt when the alert is still firing.
	payload := `[{"ruleTitle":"HighCPU","state":2,"startsAt":"2026-06-11T04:32:02.027Z","endsAt":"0001-01-01T00:00:00Z","value":95.5}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)
	require.Len(t, data.FiringAlerts, 1)
	ev := data.FiringAlerts[0]
	assert.Equal(t, "HighCPU", ev.RuleTitle)
	// "2026-06-11T04:32:02.027Z" should be converted to its Unix millisecond value
	assert.Greater(t, ev.StartsAt, int64(0))
	// "0001-01-01T00:00:00Z" (Go zero time) should map to 0
	assert.Equal(t, int64(0), ev.EndsAt)
}

func TestBuildNotificationDataFromPayload_ObjectFormat_ISO8601(t *testing.T) {
	// Verifies that a NotificationData object payload with ISO 8601 timestamps in alerts is parsed correctly.
	payload := `{
		"status": "firing",
		"alerts": [{"ruleTitle":"LowDisk","state":2,"startsAt":"2026-06-11T04:32:02.027Z","endsAt":"0001-01-01T00:00:00Z"}]
	}`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)
	require.Len(t, data.FiringAlerts, 1)
	ev := data.FiringAlerts[0]
	assert.Equal(t, "LowDisk", ev.RuleTitle)
	assert.Greater(t, ev.StartsAt, int64(0))
	assert.Equal(t, int64(0), ev.EndsAt)
}

// ─── Alertmanager format (status string + map[string]string labels) ───────────

func TestBuildNotificationDataFromPayload_AlertmanagerArrayFormat(t *testing.T) {
	// Verifies the core bug fix: Alertmanager sends "status":"firing" (string) with
	// map[string]string labels. Before the fix, state defaulted to 0 and the alert
	// was never classified into FiringAlerts.
	payload := `[{
		"status": "firing",
		"labels": {"alertname": "HighCPU", "instance": "web-01", "severity": "critical"},
		"annotations": {"summary": "CPU usage is above 90%", "description": "web-01 CPU at 95%"},
		"startsAt": "2026-06-11T04:32:02.027Z",
		"endsAt": "0001-01-01T00:00:00Z",
		"generatorURL": "/alerts/HighCPU"
	}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)
	require.Len(t, data.FiringAlerts, 1, "alert with status=firing must be in FiringAlerts")
	assert.Empty(t, data.ResolvedAlerts)

	ev := data.FiringAlerts[0]
	// Verify labels are accessible as a plain map (dot-notation in templates)
	assert.Equal(t, "HighCPU", ev.Labels["alertname"])
	assert.Equal(t, "web-01", ev.Labels["instance"])
	// Verify annotations are accessible as a plain map
	assert.Equal(t, "CPU usage is above 90%", ev.Annotations["summary"])
	// Verify common labels computed from the single alert
	assert.Equal(t, "HighCPU", data.CommonLabels["alertname"])
	// Verify State was back-filled from the "firing" status string
	assert.Equal(t, model.AlertEventFiring, ev.State)
	// Verify startsAt was parsed from ISO 8601
	assert.Greater(t, ev.StartsAt, int64(0))
	assert.Equal(t, int64(0), ev.EndsAt)
}

func TestBuildNotificationDataFromPayload_AlertmanagerArrayFormat_Pending(t *testing.T) {
	// Verifies that Alertmanager "status":"pending" maps to AlertEventPending (not Resolved).
	payload := `[{"status":"pending","labels":{"alertname":"SlowDisk"}}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "firing", data.Status)
	require.Len(t, data.FiringAlerts, 1, "pending alert must be in FiringAlerts")
	assert.Equal(t, model.AlertEventPending, data.FiringAlerts[0].State)
}

func TestBuildNotificationDataFromPayload_AlertmanagerArrayFormat_Resolved(t *testing.T) {
	// Verifies that Alertmanager "status":"resolved" maps to ResolvedAlerts.
	payload := `[{
		"status": "resolved",
		"labels": {"alertname": "HighCPU"},
		"annotations": {"summary": "CPU back to normal"}
	}]`
	data, err := BuildNotificationDataFromPayload(payload)
	require.NoError(t, err)
	assert.Equal(t, "resolved", data.Status)
	assert.Empty(t, data.FiringAlerts)
	require.Len(t, data.ResolvedAlerts, 1)
	assert.Equal(t, model.AlertEventResolved, data.ResolvedAlerts[0].State)
}

func TestRenderer_AlertmanagerAnnotationsDotNotation(t *testing.T) {
	// Verifies the end-to-end fix: template can use .Annotations.summary via dot-notation
	// when the payload uses Alertmanager format (map[string]string annotations).
	r := NewRenderer()
	data := &NotificationData{
		Status: "firing",
		FiringAlerts: []TemplateAlert{
			{
				RuleTitle:   "HighCPU",
				Status:      "firing",
				State:       model.AlertEventFiring,
				Labels:      map[string]string{"alertname": "HighCPU", "instance": "web-01"},
				Annotations: map[string]string{"summary": "CPU is too high"},
			},
		},
	}
	// This is the exact template pattern that was failing before the fix
	result, err := r.Render(`{{ range .FiringAlerts }}{{ .Annotations.summary }}{{ end }}`, data)
	require.NoError(t, err)
	assert.Equal(t, "CPU is too high", result)
}
