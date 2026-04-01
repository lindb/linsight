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
	html "html/template"
	"strings"
	"text/template"
	"time"

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
	Alerts            []*model.AlertEvent       // all alerts in the group
	FiringAlerts      []*model.AlertEvent       // currently firing alerts
	ResolvedAlerts    []*model.AlertEvent        // recently resolved alerts
	CommonLabels      map[string]string          // labels shared by all alerts
	CommonAnnotations map[string]string          // annotations shared by all alerts
	GroupLabels       map[string]string          // labels used to form this group
	ReceiverName      string                     // name of the target receiver
	Status            string                     // "firing" or "resolved"
	ExternalURL       string                     // base URL for generating links
}

// Renderer provides Go template rendering for notifications.
type Renderer interface {
	// RenderText renders a text/template string with the given data.
	RenderText(tmplStr string, data *NotificationData) (string, error)
	// RenderHTML renders an html/template string with the given data (for email bodies).
	RenderHTML(tmplStr string, data *NotificationData) (string, error)
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

func (r *renderer) RenderText(tmplStr string, data *NotificationData) (string, error) {
	return renderText(tmplStr, data)
}

func (r *renderer) RenderHTML(tmplStr string, data *NotificationData) (string, error) {
	return renderHTML(tmplStr, data)
}

func (r *renderer) RenderDefaultTitle(data *NotificationData) (string, error) {
	return renderText(defaultTitleTemplate, data)
}

func (r *renderer) RenderDefaultBody(data *NotificationData) (string, error) {
	return renderText(defaultBodyTemplate, data)
}

// RenderWithContext renders a template string respecting context cancellation.
func RenderWithContext(ctx context.Context, r Renderer, tmplType model.TemplateType, tmplStr string, data *NotificationData) (string, error) {
	resultCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		var result string
		var err error
		switch tmplType {
		case model.TemplateHTMLType:
			result, err = r.RenderHTML(tmplStr, data)
		default:
			result, err = r.RenderText(tmplStr, data)
		}
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

// templateFuncMap returns template helper functions shared across all templates.
func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		// upper converts a string to upper case
		"upper": strings.ToUpper,
		// lower converts a string to lower case
		"lower": strings.ToLower,
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
		// annotationsGet retrieves a single annotation value by key
		"annotationsGet": func(annotationsJSON interface{}, key string) string {
			m := parseJSONMapFromAny(annotationsJSON)
			return m[key]
		},
	}
}

// htmlTemplateFuncMap adapts the text funcmap for use with html/template.
func htmlTemplateFuncMap() html.FuncMap {
	tf := templateFuncMap()
	hf := make(html.FuncMap, len(tf))
	for k, v := range tf {
		hf[k] = v
	}
	return hf
}

func renderText(tmplStr string, data *NotificationData) (string, error) {
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

func renderHTML(tmplStr string, data *NotificationData) (string, error) {
	tmpl, err := html.New("notification").Funcs(htmlTemplateFuncMap()).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse html template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute html template: %w", err)
	}
	return buf.String(), nil
}
