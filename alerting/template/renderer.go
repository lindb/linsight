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
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/linsight/model"
)

const (
	// defaultBodyTemplate is used when no custom template is configured.
	defaultBodyTemplate = `{{ if eq .Status "firing" }}🔥 FIRING{{ else }}✅ RESOLVED{{ end }} - {{ .ReceiverName }}
{{ range .FiringAlerts }}
[{{ .RuleTitle }}] {{ if .Annotations }}{{ annotationsGet .Annotations "summary" }}{{ end }}
Labels: {{ labelsFormat .Labels }}
Value: {{ .Value }}
{{ end }}
{{ range .ResolvedAlerts }}
[RESOLVED] {{ .RuleTitle }}
{{ end }}`

	// defaultTitleTemplate is used for notification titles when no custom template is configured.
	defaultTitleTemplate = `[{{ .Status | upper }}] {{ .CommonLabels.alertname }}{{ if .CommonLabels.instance }} ({{ .CommonLabels.instance }}){{ end }}`
)

// NotificationData contains the data injected into notification templates.
type NotificationData struct {
	Alerts            []*model.AlertEvent `json:"alerts"`            // all alerts in the group
	FiringAlerts      []*model.AlertEvent `json:"firingAlerts"`      // currently firing alerts
	ResolvedAlerts    []*model.AlertEvent `json:"resolvedAlerts"`    // recently resolved alerts
	CommonLabels      map[string]string   `json:"commonLabels"`      // labels shared by all alerts
	CommonAnnotations map[string]string   `json:"commonAnnotations"` // annotations shared by all alerts
	GroupLabels       map[string]string   `json:"groupLabels"`       // labels used to form this group
	ReceiverName      string              `json:"receiverName"`      // name of the target receiver
	Status            string              `json:"status"`            // "firing" or "resolved"
	ExternalURL       string              `json:"externalURL"`       // base URL for generating links
}

// Renderer provides Go template rendering for notifications.
type Renderer interface {
	// Render renders a text/template string with the given data.
	Render(tmplStr string, data *NotificationData) (string, error)
	// RenderDefaultTitle renders the built-in title template.
	RenderDefaultTitle(data *NotificationData) (string, error)
	// RenderDefaultBody renders the built-in body template.
	RenderDefaultBody(data *NotificationData) (string, error)
}

type renderer struct{}

// NewRenderer creates a Renderer instance.
func NewRenderer() Renderer {
	return &renderer{}
}

func (r *renderer) Render(tmplStr string, data *NotificationData) (string, error) {
	return renderText(tmplStr, data)
}

func (r *renderer) RenderDefaultTitle(data *NotificationData) (string, error) {
	return renderText(defaultTitleTemplate, data)
}

func (r *renderer) RenderDefaultBody(data *NotificationData) (string, error) {
	return renderText(defaultBodyTemplate, data)
}

// RenderWithContext renders a template string respecting context cancellation.
func RenderWithContext(ctx context.Context, r Renderer, tmplStr string, data *NotificationData) (string, error) {
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		result, err := r.Render(tmplStr, data)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		return "", fmt.Errorf("template rendering timed out: %w", ctx.Err())
	}
}

// BuildNotificationData constructs NotificationData from a list of AlertEvents.
func BuildNotificationData(events []*model.AlertEvent, receiverName, status, externalURL string, groupLabels map[string]string) *NotificationData {
	data := &NotificationData{
		Alerts:            events,
		CommonLabels:      make(map[string]string),
		CommonAnnotations: make(map[string]string),
		GroupLabels:       groupLabels,
		ReceiverName:      receiverName,
		Status:            status,
		ExternalURL:       externalURL,
	}

	for _, ev := range events {
		if ev.State == model.AlertEventFiring || ev.State == model.AlertEventPending {
			data.FiringAlerts = append(data.FiringAlerts, ev)
		} else if ev.State == model.AlertEventResolved {
			data.ResolvedAlerts = append(data.ResolvedAlerts, ev)
		}
	}

	// compute common labels across all alerts
	data.CommonLabels = computeCommonLabels(events)
	data.CommonAnnotations = computeCommonAnnotations(events)

	return data
}

// BuildNotificationDataFromPayload deserializes example payload JSON into NotificationData.
// When FiringAlerts/ResolvedAlerts are absent but Alerts is non-empty, it auto-derives them
// by inspecting each AlertEvent.State. CommonLabels and CommonAnnotations are also derived
// from the Alerts list when they are not provided.
func BuildNotificationDataFromPayload(payloadJSON string) (*NotificationData, error) {
	if payloadJSON == "" {
		return &NotificationData{}, nil
	}
	data := &NotificationData{}
	if err := encoding.JSONUnmarshal([]byte(payloadJSON), data); err != nil {
		return nil, fmt.Errorf("parse notification payload: %w", err)
	}

	// auto-derive sub-lists when not explicitly provided in the payload
	if len(data.FiringAlerts) == 0 && len(data.ResolvedAlerts) == 0 && len(data.Alerts) > 0 {
		for _, ev := range data.Alerts {
			if ev.State == model.AlertEventFiring || ev.State == model.AlertEventPending {
				data.FiringAlerts = append(data.FiringAlerts, ev)
			} else if ev.State == model.AlertEventResolved {
				data.ResolvedAlerts = append(data.ResolvedAlerts, ev)
			}
		}
	}

	// auto-derive common labels/annotations when not set
	if len(data.CommonLabels) == 0 && len(data.Alerts) > 0 {
		data.CommonLabels = computeCommonLabels(data.Alerts)
	}
	if len(data.CommonAnnotations) == 0 && len(data.Alerts) > 0 {
		data.CommonAnnotations = computeCommonAnnotations(data.Alerts)
	}

	return data, nil
}

// computeCommonLabels returns labels that are identical across all given events.
func computeCommonLabels(events []*model.AlertEvent) map[string]string {
	if len(events) == 0 {
		return nil
	}

	// seed from first event
	first := parseJSONMap(events[0].Labels)
	common := make(map[string]string, len(first))
	for k, v := range first {
		common[k] = v
	}

	for _, ev := range events[1:] {
		labels := parseJSONMap(ev.Labels)
		for k := range common {
			if labels[k] != common[k] {
				delete(common, k)
			}
		}
	}
	return common
}

// computeCommonAnnotations returns annotations shared by all events.
func computeCommonAnnotations(events []*model.AlertEvent) map[string]string {
	if len(events) == 0 {
		return nil
	}
	first := parseJSONMap(events[0].Annotations)
	common := make(map[string]string, len(first))
	for k, v := range first {
		common[k] = v
	}

	for _, ev := range events[1:] {
		annotations := parseJSONMap(ev.Annotations)
		for k := range common {
			if annotations[k] != common[k] {
				delete(common, k)
			}
		}
	}
	return common
}

// toTitleCase capitalises the first letter of each word.
func toTitleCase(s string) string {
	prevSpace := true
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			prevSpace = true
			return r
		}
		if prevSpace {
			prevSpace = false
			return unicode.ToTitle(r)
		}
		return r
	}, s)
}

// templateFuncMap returns template helper functions shared across all templates.
func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		// upper converts a string to upper case
		"upper": strings.ToUpper,
		// lower converts a string to lower case
		"lower": strings.ToLower,
		// title converts a string to title case (first letter of each word capitalised)
		"title": toTitleCase,
		// trimSpace removes leading and trailing white space from a string
		"trimSpace": strings.TrimSpace,
		// join concatenates string slice elements separated by sep
		"join": func(sep string, elems []string) string {
			return strings.Join(elems, sep)
		},
		// match reports whether the string s contains any match for the regexp pattern
		"match": func(pattern, s string) bool {
			ok, _ := regexp.MatchString(pattern, s)
			return ok
		},
		// reReplaceAll returns a copy of s with all matches of pattern replaced by repl
		"reReplaceAll": func(pattern, repl, s string) string {
			return regexp.MustCompile(pattern).ReplaceAllString(s, repl)
		},
		// humanizeDuration converts a duration in milliseconds to a human-readable string (e.g. "2h3m4s")
		"humanizeDuration": func(ms int64) string {
			d := time.Duration(ms) * time.Millisecond
			d = d.Round(time.Second)
			h := int(d.Hours())
			m := int(d.Minutes()) % 60
			s := int(d.Seconds()) % 60
			if h > 0 {
				return fmt.Sprintf("%dh%dm%ds", h, m, s)
			}
			if m > 0 {
				return fmt.Sprintf("%dm%ds", m, s)
			}
			return fmt.Sprintf("%ds", s)
		},
		// timeFormat formats a unix millisecond timestamp
		"timeFormat": func(ms int64) string {
			return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
		},
		// labelsFormat formats a datatypes.JSON label map as a compact string
		"labelsFormat": func(labelsJSON interface{}) string {
			m := parseJSONMapFromAny(labelsJSON)
			parts := make([]string, 0, len(m))
			for k, v := range m {
				parts = append(parts, fmt.Sprintf("%s=%q", k, v))
			}
			return "{" + strings.Join(parts, ", ") + "}"
		},
		// labelsGet retrieves a single label value by key
		"labelsGet": func(labelsJSON interface{}, key string) string {
			m := parseJSONMapFromAny(labelsJSON)
			return m[key]
		},
		// annotationsGet retrieves a single annotation value by key
		"annotationsGet": func(annotationsJSON interface{}, key string) string {
			m := parseJSONMapFromAny(annotationsJSON)
			return m[key]
		},
	}
}

func renderText(tmplStr string, data *NotificationData) (string, error) {
	// Use ParseFiles-style approach: assign template name so {{define}} blocks register correctly.
	// Using template.New + Parse lets root content and any {{define}} blocks coexist in one string.
	tmpl, err := template.New("notification").Funcs(templateFuncMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
