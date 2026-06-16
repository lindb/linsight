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

// TemplateAlert is a template-friendly representation of a single alert event.
// Labels and Annotations use plain string maps so templates can access them with
// direct dot-notation (e.g. .Labels.alertname, .Annotations.summary) without
// requiring the annotationsGet/labelsGet helper functions.
type TemplateAlert struct {
	UID          string                `json:"uid"`
	RuleUID      string                `json:"ruleUID"`
	RuleTitle    string                `json:"ruleTitle"`
	Fingerprint  string                `json:"fingerprint"`
	Status       string                `json:"status"`       // "firing" or "resolved"
	State        model.AlertEventState `json:"state"`        // numeric state kept for backward compat
	Severity     model.Severity        `json:"severity"`
	Labels       map[string]string     `json:"labels"`
	Annotations  map[string]string     `json:"annotations"`
	Value        float64               `json:"value"`
	StartsAt     int64                 `json:"startsAt"`     // unix milliseconds
	EndsAt       int64                 `json:"endsAt"`       // unix milliseconds
	PendingSince int64                 `json:"pendingSince"` // unix milliseconds
	NotifiedAt   int64                 `json:"notifiedAt"`   // unix milliseconds
	GeneratorURL string                `json:"generatorURL"`
}

// NotificationData contains the data injected into notification templates.
type NotificationData struct {
	Alerts            []TemplateAlert   `json:"alerts"`            // all alerts in the group
	FiringAlerts      []TemplateAlert   `json:"firingAlerts"`      // currently firing alerts
	ResolvedAlerts    []TemplateAlert   `json:"resolvedAlerts"`    // recently resolved alerts
	CommonLabels      map[string]string `json:"commonLabels"`      // labels shared by all alerts
	CommonAnnotations map[string]string `json:"commonAnnotations"` // annotations shared by all alerts
	GroupLabels       map[string]string `json:"groupLabels"`       // labels used to form this group
	ReceiverName      string            `json:"receiverName"`      // name of the target receiver
	Status            string            `json:"status"`            // "firing" or "resolved"
	ExternalURL       string            `json:"externalURL"`       // base URL for generating links
	// OrgTemplates holds the merged content of all org-level notification templates.
	// It is populated by the alert engine before calling Notify so that notifiers can
	// resolve {{define "NAME"}} named blocks via RenderNamed without changing the Notifier interface.
	OrgTemplates string `json:"-"`
}

// Renderer provides Go template rendering for notifications.
type Renderer interface {
	// Render renders a text/template string with the given data.
	Render(tmplStr string, data *NotificationData) (string, error)
	// RenderNamed parses orgTemplates (which may contain multiple {{define "NAME"}} blocks)
	// and executes the named block identified by name.
	RenderNamed(orgTemplates, name string, data *NotificationData) (string, error)
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

// alertEventToTemplateAlert converts a model.AlertEvent (DB layer) to a TemplateAlert
// (template layer). Labels and Annotations are decoded from datatypes.JSON to plain maps
// so templates can access them with direct dot-notation.
func alertEventToTemplateAlert(ev *model.AlertEvent) TemplateAlert {
	status := "resolved"
	if ev.State == model.AlertEventFiring || ev.State == model.AlertEventPending {
		status = "firing"
	}
	return TemplateAlert{
		UID:          ev.UID,
		RuleUID:      ev.RuleUID,
		RuleTitle:    ev.RuleTitle,
		Fingerprint:  ev.Fingerprint,
		Status:       status,
		State:        ev.State,
		Severity:     ev.Severity,
		Labels:       parseJSONMap(ev.Labels),
		Annotations:  parseJSONMap(ev.Annotations),
		Value:        ev.Value,
		StartsAt:     ev.StartsAt,
		EndsAt:       ev.EndsAt,
		PendingSince: ev.PendingSince,
		NotifiedAt:   ev.NotifiedAt,
		GeneratorURL: ev.GeneratorURL,
	}
}

func (r *renderer) Render(tmplStr string, data *NotificationData) (string, error) {
	return renderText(tmplStr, data)
}

func (r *renderer) RenderNamed(orgTemplates, name string, data *NotificationData) (string, error) {
	// Parse all org templates so every {{define}} block is registered as a named sub-template.
	tmpl, err := template.New("root").Funcs(templateFuncMap()).Parse(orgTemplates)
	if err != nil {
		return "", fmt.Errorf("parse org templates: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("execute named template %q: %w", name, err)
	}
	return buf.String(), nil
}

func (r *renderer) RenderDefaultTitle(data *NotificationData) (string, error) {
	return renderText(defaultTitleTemplate, data)
}

func (r *renderer) RenderDefaultBody(data *NotificationData) (string, error) {
	return renderText(defaultBodyTemplate, data)
}

// BuildNotificationData constructs NotificationData from a list of AlertEvents.
func BuildNotificationData(events []*model.AlertEvent, receiverName, status, externalURL string, groupLabels map[string]string) *NotificationData {
	// Convert DB-layer AlertEvents to template-friendly TemplateAlerts up front so that
	// all downstream template code works with plain map[string]string labels/annotations.
	alerts := make([]TemplateAlert, len(events))
	for i, ev := range events {
		alerts[i] = alertEventToTemplateAlert(ev)
	}
	return buildFromTemplateAlerts(alerts, receiverName, status, externalURL, groupLabels)
}

// buildFromTemplateAlerts constructs NotificationData directly from pre-converted TemplateAlerts.
// It is used by both BuildNotificationData and BuildNotificationDataFromPayload.
func buildFromTemplateAlerts(alerts []TemplateAlert, receiverName, status, externalURL string, groupLabels map[string]string) *NotificationData {
	data := &NotificationData{
		Alerts:            alerts,
		CommonLabels:      make(map[string]string),
		CommonAnnotations: make(map[string]string),
		GroupLabels:       groupLabels,
		ReceiverName:      receiverName,
		Status:            status,
		ExternalURL:       externalURL,
	}

	for _, ta := range alerts {
		switch ta.State {
		// NoData and Error are operationally "active" conditions that need attention,
		// so they belong in FiringAlerts alongside Firing and Pending.
		case model.AlertEventFiring, model.AlertEventPending,
			model.AlertEventNoData, model.AlertEventError:
			data.FiringAlerts = append(data.FiringAlerts, ta)
		case model.AlertEventResolved:
			data.ResolvedAlerts = append(data.ResolvedAlerts, ta)
		}
	}

	data.CommonLabels = computeCommonLabels(alerts)
	data.CommonAnnotations = computeCommonAnnotations(alerts)

	return data
}

// BuildNotificationDataFromPayload deserializes example payload JSON into NotificationData.
// The payload may be either:
//   - A JSON object matching NotificationData (full or partial). When FiringAlerts/ResolvedAlerts
//     are absent but Alerts is non-empty, they are auto-derived from each TemplateAlert.State.
//   - A JSON array of alert objects, supporting both Alertmanager format
//     ("status":"firing" + map[string]string labels) and LinSight internal format
//     ("state":2 int enum + map[string]string labels). In this case NotificationData
//     fields are derived automatically.
//
// Timestamp fields (startsAt, endsAt, pendingSince, notifiedAt) accept both Unix millisecond
// integers and ISO 8601 / RFC3339 strings (e.g. "2026-06-11T04:32:02.027Z").
// CommonLabels and CommonAnnotations are always derived from Alerts when not provided.
func BuildNotificationDataFromPayload(payloadJSON string) (*NotificationData, error) {
	if payloadJSON == "" {
		return &NotificationData{}, nil
	}

	// Detect JSON shape: a leading '[' means the caller sent an alert array directly;
	// anything else ('{' or 'n' for null) is treated as a NotificationData object.
	if strings.HasPrefix(strings.TrimSpace(payloadJSON), "[") {
		var payloads []*templateAlertPayload
		if err := encoding.JSONUnmarshal([]byte(payloadJSON), &payloads); err != nil {
			return nil, fmt.Errorf("parse notification payload: %w", err)
		}
		alerts := convertAlertPayloads(payloads)
		// Derive overall status from the alert states.
		status := "resolved"
		for _, ta := range alerts {
			if ta.State == model.AlertEventFiring || ta.State == model.AlertEventPending {
				status = "firing"
				break
			}
		}
		return buildFromTemplateAlerts(alerts, "", status, "", nil), nil
	}

	ndp := &notificationDataPayload{}
	if err := encoding.JSONUnmarshal([]byte(payloadJSON), ndp); err != nil {
		return nil, fmt.Errorf("parse notification payload: %w", err)
	}
	data := ndp.toNotificationData()

	// auto-derive sub-lists when not explicitly provided in the payload
	if len(data.FiringAlerts) == 0 && len(data.ResolvedAlerts) == 0 && len(data.Alerts) > 0 {
		for _, ta := range data.Alerts {
			switch ta.State {
			case model.AlertEventFiring, model.AlertEventPending,
				model.AlertEventNoData, model.AlertEventError:
				data.FiringAlerts = append(data.FiringAlerts, ta)
			case model.AlertEventResolved:
				data.ResolvedAlerts = append(data.ResolvedAlerts, ta)
			}
		}
	}

	// auto-derive top-level status when not explicitly set in the payload
	if data.Status == "" && len(data.Alerts) > 0 {
		data.Status = "resolved"
		for _, ta := range data.Alerts {
			if ta.State == model.AlertEventFiring || ta.State == model.AlertEventPending ||
				ta.State == model.AlertEventNoData || ta.State == model.AlertEventError {
				data.Status = "firing"
				break
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

// computeCommonLabels returns labels that are identical across all given alerts.
// Returns an empty map (never nil) so callers can range over it safely.
func computeCommonLabels(alerts []TemplateAlert) map[string]string {
	if len(alerts) == 0 {
		return make(map[string]string)
	}

	// seed from first alert; Labels is already map[string]string, no JSON decoding needed
	common := make(map[string]string, len(alerts[0].Labels))
	for k, v := range alerts[0].Labels {
		common[k] = v
	}

	for _, ta := range alerts[1:] {
		for k := range common {
			if ta.Labels[k] != common[k] {
				delete(common, k)
			}
		}
	}
	return common
}

// computeCommonAnnotations returns annotations shared by all alerts.
// Returns an empty map (never nil) so callers can range over it safely.
func computeCommonAnnotations(alerts []TemplateAlert) map[string]string {
	if len(alerts) == 0 {
		return make(map[string]string)
	}

	// seed from first alert; Annotations is already map[string]string, no JSON decoding needed
	common := make(map[string]string, len(alerts[0].Annotations))
	for k, v := range alerts[0].Annotations {
		common[k] = v
	}

	for _, ta := range alerts[1:] {
		for k := range common {
			if ta.Annotations[k] != common[k] {
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
		// reReplaceAll returns a copy of s with all matches of pattern replaced by repl;
		// returns s unchanged when pattern is not a valid regular expression.
		"reReplaceAll": func(pattern, repl, s string) string {
			re, err := regexp.Compile(pattern)
			if err != nil {
				return s
			}
			return re.ReplaceAllString(s, repl)
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
		// labelsFormat formats a label map as a compact string
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

// defineNameRe matches {{define "name"}} and {{- define `name`}} patterns (both double-quoted
// and backtick-quoted names) to extract named block names in source order.
// Templates() has no order guarantee, so regex over source is the only reliable approach.
// Character class ["` + "`" + `] matches either delimiter; \x60 is the backtick (ASCII 96).
var defineNameRe = regexp.MustCompile("\\{\\{-?\\s*define\\s+[\"\\x60]([^\"\\x60]+)[\"\\x60]")

// renderWithOrgBlocks executes tmplStr with org-level {{define}} blocks available as
// fallbacks. tmplStr's own definitions take precedence over same-named org blocks,
// preventing "multiple definition" errors when previewing a template that is already
// stored in the org template library.
//
// When tmplStr contains only {{define}} blocks (no root execution content), the first
// user-defined block is auto-executed so that previewing org-template content — which
// is stored entirely as named blocks — always produces visible output.
func renderWithOrgBlocks(orgTemplates, tmplStr string, data *NotificationData) (string, error) {
	funcs := templateFuncMap()

	// Extract user-defined block names in source order before parsing, so the
	// auto-execute fallback below honours the order the user wrote them.
	var userDefinedNames []string
	for _, m := range defineNameRe.FindAllStringSubmatch(tmplStr, -1) {
		userDefinedNames = append(userDefinedNames, m[1])
	}

	// Parse user template first — it becomes the root execution context and its
	// {{define}} blocks take priority over any same-named blocks in the org templates.
	tmpl, err := template.New("notification").Funcs(funcs).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	if strings.TrimSpace(orgTemplates) != "" {
		orgTmpl, orgErr := template.New("org").Funcs(funcs).Parse(orgTemplates)
		if orgErr != nil {
			return "", fmt.Errorf("parse org templates: %w", orgErr)
		}
		// Inject org blocks as fallbacks; skip any already defined in the user template
		// to avoid "multiple definition" panics when editing an existing org template.
		for _, t := range orgTmpl.Templates() {
			name := t.Name()
			if name == "org" || t.Tree == nil || t.Tree.Root == nil {
				continue
			}
			if tmpl.Lookup(name) == nil {
				if _, addErr := tmpl.AddParseTree(name, t.Tree); addErr != nil {
					return "", fmt.Errorf("merge org template %q: %w", name, addErr)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	// When the user template is composed entirely of {{define}} blocks (a common
	// pattern for org-level templates), the root body is empty and Execute produces
	// no output. Auto-execute the first user-defined block so the preview is useful.
	if strings.TrimSpace(buf.String()) == "" && len(userDefinedNames) > 0 {
		buf.Reset()
		if err := tmpl.ExecuteTemplate(&buf, userDefinedNames[0], data); err != nil {
			return "", fmt.Errorf("execute template: %w", err)
		}
	}

	return buf.String(), nil
}
