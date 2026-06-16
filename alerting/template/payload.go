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
	"time"

	"github.com/lindb/linsight/model"
)

// templateAlertPayload is the deserialization target for a single alert object inside a preview
// request payload. It deliberately supports two wire formats in one struct:
//
//   - LinSight internal format: uses "state" (int enum, e.g. 2 for firing) and
//     optionally omits "status".
//   - Alertmanager format: uses "status" ("firing" / "resolved") and may omit "state"
//     entirely (defaults to 0).
//
// After unmarshalling, toTemplateAlert() reconciles the two fields so that downstream
// classification code can rely on State exclusively.
//
// Labels and Annotations are plain map[string]string — both formats send them that way,
// and it avoids the datatypes.JSON / []byte incompatibility with Go template dot-notation.
//
// Timestamp fields use *time.Time so the standard JSON decoder can parse ISO 8601 strings
// like "2026-06-11T04:32:02.027Z" (Alertmanager convention). Nil / zero time maps to 0.
type templateAlertPayload struct {
	UID          string                `json:"uid"`
	RuleUID      string                `json:"ruleUID"`
	RuleTitle    string                `json:"ruleTitle"`
	Fingerprint  string                `json:"fingerprint"`
	Status       string                `json:"status"`      // Alertmanager: "firing" / "resolved"
	State        model.AlertEventState `json:"state"`       // LinSight int fallback (0 = unset)
	Severity     model.Severity        `json:"severity"`
	Labels       map[string]string     `json:"labels"`
	Annotations  map[string]string     `json:"annotations"`
	Value        float64               `json:"value"`
	StartsAt     *time.Time            `json:"startsAt"`
	EndsAt       *time.Time            `json:"endsAt"`
	PendingSince *time.Time            `json:"pendingSince"`
	NotifiedAt   *time.Time            `json:"notifiedAt"`
	GeneratorURL string                `json:"generatorURL"`
}

// toTemplateAlert converts a payload into a TemplateAlert, reconciling the dual-format
// Status (string) / State (int) fields:
//
//   - Alertmanager path: Status is set, State is 0 → back-fill State from Status string so
//     downstream classification (FiringAlerts / ResolvedAlerts split) works correctly.
//   - LinSight path: State is set, Status is empty → derive the Status string from State so
//     templates can use {{ .Status }} without checking State numerically.
//   - Both set → trust each field as-is (caller provided explicit values).
//   - Neither set → default to "resolved" / AlertEventResolved.
func (p *templateAlertPayload) toTemplateAlert() TemplateAlert {
	status := p.Status
	state := p.State

	switch {
	case status != "" && state == 0:
		// Alertmanager format: back-fill numeric State from the string status.
		switch status {
		case "firing":
			state = model.AlertEventFiring
		case "pending":
			state = model.AlertEventPending
		default:
			state = model.AlertEventResolved
		}
	case status == "" && state != 0:
		// LinSight format: derive readable Status from numeric State.
		if state == model.AlertEventFiring || state == model.AlertEventPending {
			status = "firing"
		} else {
			status = "resolved"
		}
	case status == "" && state == 0:
		// Neither field provided — treat as resolved.
		status = "resolved"
		state = model.AlertEventResolved
	}

	return TemplateAlert{
		UID:          p.UID,
		RuleUID:      p.RuleUID,
		RuleTitle:    p.RuleTitle,
		Fingerprint:  p.Fingerprint,
		Status:       status,
		State:        state,
		Severity:     p.Severity,
		Labels:       p.Labels,
		Annotations:  p.Annotations,
		Value:        p.Value,
		StartsAt:     toMillis(p.StartsAt),
		EndsAt:       toMillis(p.EndsAt),
		PendingSince: toMillis(p.PendingSince),
		NotifiedAt:   toMillis(p.NotifiedAt),
		GeneratorURL: p.GeneratorURL,
	}
}

// notificationDataPayload mirrors NotificationData for payload deserialization,
// using []*templateAlertPayload slices so that alert events with ISO 8601 timestamps
// and either Alertmanager or LinSight format can be decoded without error.
type notificationDataPayload struct {
	Alerts            []*templateAlertPayload `json:"alerts"`
	FiringAlerts      []*templateAlertPayload `json:"firingAlerts"`
	ResolvedAlerts    []*templateAlertPayload `json:"resolvedAlerts"`
	CommonLabels      map[string]string       `json:"commonLabels"`
	CommonAnnotations map[string]string       `json:"commonAnnotations"`
	GroupLabels       map[string]string       `json:"groupLabels"`
	ReceiverName      string                  `json:"receiverName"`
	Status            string                  `json:"status"`
	ExternalURL       string                  `json:"externalURL"`
}

// toMillis converts a *time.Time pointer to Unix milliseconds.
// A nil pointer or the Go zero time ("0001-01-01T00:00:00Z") returns 0.
func toMillis(t *time.Time) int64 {
	if t == nil || t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}

// convertAlertPayloads converts a slice of templateAlertPayload to []TemplateAlert.
// Nil elements in the slice (which JSON arrays can produce with explicit null values)
// are silently skipped to avoid a nil-pointer dereference.
func convertAlertPayloads(payloads []*templateAlertPayload) []TemplateAlert {
	if len(payloads) == 0 {
		return nil
	}
	alerts := make([]TemplateAlert, 0, len(payloads))
	for _, p := range payloads {
		if p == nil {
			continue
		}
		alerts = append(alerts, p.toTemplateAlert())
	}
	return alerts
}

// toNotificationData converts the payload struct to a NotificationData value.
// Alert slices are converted to []TemplateAlert with string labels/annotations ready
// for direct template dot-notation access (.Labels.alertname, .Annotations.summary).
func (p *notificationDataPayload) toNotificationData() *NotificationData {
	return &NotificationData{
		Alerts:            convertAlertPayloads(p.Alerts),
		FiringAlerts:      convertAlertPayloads(p.FiringAlerts),
		ResolvedAlerts:    convertAlertPayloads(p.ResolvedAlerts),
		CommonLabels:      p.CommonLabels,
		CommonAnnotations: p.CommonAnnotations,
		GroupLabels:       p.GroupLabels,
		ReceiverName:      p.ReceiverName,
		Status:            p.Status,
		ExternalURL:       p.ExternalURL,
	}
}
