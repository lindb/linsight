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

package slack

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/slack-go/slack"

	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

//go:embed schema.json
var schema string

func init() {
	types.Register(&types.NotifierDef{
		Plugin: &model.NotifierPlugin{
			Type:         "Slack",
			DisplayName:  "Slack",
			Description:  "Send alert notifications via Slack",
			ConfigSchema: json.RawMessage(schema),
		},
		NewFn:      newSlackNotifier,
		SummaryKey: "channel", // show target Slack channel in the contact-point overview
	})
}

// Setting holds global Slack plugin configuration (bot token).
type Setting struct {
	Token string `json:"token"`
}

// Config holds per-channel Slack configuration.
type Config struct {
	Channel string `json:"channel"`
	// Optional named template names ({{define "NAME"}} blocks from org templates).
	// Empty value falls back to the built-in defaults.
	Title  string `json:"title"`
	Body   string `json:"body"`
	Footer string `json:"footer"`
}

type notifier struct {
	setting Setting
	cfg     Config
	cli     *slack.Client
	engine  *template.RenderingEngine
	logger  logger.Logger
}

func newSlackNotifier(settingJSON, cfgJSON json.RawMessage, engine *template.RenderingEngine) (types.Notifier, error) {
	var settingData Setting
	if err := encoding.JSONUnmarshal([]byte(settingJSON), &settingData); err != nil {
		return nil, err
	}
	var cfgData Config
	if err := encoding.JSONUnmarshal([]byte(cfgJSON), &cfgData); err != nil {
		return nil, err
	}
	return &notifier{
		setting: settingData,
		cfg:     cfgData,
		cli:     slack.New(settingData.Token),
		engine:  engine,
		logger:  logger.GetLogger("Alerting", "SlackNotifier"),
	}, nil
}

func (n *notifier) Notify(ctx context.Context, data *template.NotificationData) error {
	// renderOrDefault renders the template string via engine (with org template blocks merged in)
	// when name is non-empty; falls back to defaultFn otherwise.
	renderOrDefault := func(name string, defaultFn func() (string, error)) (string, error) {
		if name != "" && n.engine != nil {
			return n.engine.Render(data.OrgID, name, data)
		}
		return defaultFn()
	}

	// renderer is used only inside the defaultFn lambdas for built-in title/body templates.
	renderer := template.NewRenderer()

	var attachments []slack.Attachment
	for _, alert := range data.FiringAlerts {
		// Build a single-alert payload so named templates can reference .Alerts[0] etc.
		alertData := &template.NotificationData{
			OrgID:             data.OrgID,
			Alerts:            []template.TemplateAlert{alert},
			FiringAlerts:      []template.TemplateAlert{alert},
			CommonLabels:      data.CommonLabels,
			CommonAnnotations: data.CommonAnnotations,
			GroupLabels:       data.GroupLabels,
			ReceiverName:      data.ReceiverName,
			Status:            "firing",
			ExternalURL:       data.ExternalURL,
			OrgTemplates:      data.OrgTemplates,
		}

		title, err := renderOrDefault(n.cfg.Title, func() (string, error) {
			return renderer.RenderDefaultTitle(alertData)
		})
		if err != nil {
			return fmt.Errorf("render firing alert title: %w", err)
		}
		body, err := renderOrDefault(n.cfg.Body, func() (string, error) {
			return renderer.RenderDefaultBody(alertData)
		})
		if err != nil {
			return fmt.Errorf("render firing alert body: %w", err)
		}

		n.logger.Info("slack firing attachment rendered",
			logger.String("channel", n.cfg.Channel),
			logger.String("title", title),
			logger.String("body", body))

		var titleLink string
		if data.ExternalURL != "" && alert.GeneratorURL != "" {
			titleLink = data.ExternalURL + alert.GeneratorURL
		} else {
			titleLink = alert.GeneratorURL
		}

		att := slack.Attachment{
			Color:      "danger",
			Title:      title,
			TitleLink:  titleLink,
			Text:       body,
			MarkdownIn: []string{"text"},
		}
		// Only attach a timestamp when the alert has actually started; StartsAt=0 would
		// display as the 1970 epoch in Slack which is confusing.
		if alert.StartsAt > 0 {
			att.Ts = json.Number(fmt.Sprintf("%d", alert.StartsAt/1000))
		}
		if n.cfg.Footer != "" {
			footer, err := renderOrDefault(n.cfg.Footer, func() (string, error) { return "", nil })
			if err != nil {
				return fmt.Errorf("render firing alert footer: %w", err)
			}
			att.Footer = footer
		}
		attachments = append(attachments, att)
	}

	for _, alert := range data.ResolvedAlerts {
		resolvedData := &template.NotificationData{
			OrgID:             data.OrgID,
			Alerts:            []template.TemplateAlert{alert},
			ResolvedAlerts:    []template.TemplateAlert{alert},
			CommonLabels:      data.CommonLabels,
			CommonAnnotations: data.CommonAnnotations,
			GroupLabels:       data.GroupLabels,
			ReceiverName:      data.ReceiverName,
			Status:            "resolved",
			ExternalURL:       data.ExternalURL,
			OrgTemplates:      data.OrgTemplates,
		}
		title, err := renderOrDefault(n.cfg.Title, func() (string, error) {
			return renderer.RenderDefaultTitle(resolvedData)
		})
		if err != nil {
			return fmt.Errorf("render resolved alert title: %w", err)
		}

		n.logger.Info("slack resolved attachment rendered",
			logger.String("channel", n.cfg.Channel),
			logger.String("title", title))

		// Format resolution time only when it is meaningful.
		var resolvedText string
		if alert.EndsAt > 0 {
			resolvedText = fmt.Sprintf("Alert resolved at %s",
				time.UnixMilli(alert.EndsAt).Format("2006-01-02 15:04:05"))
		} else {
			resolvedText = "Alert resolved"
		}

		att := slack.Attachment{
			Color:      "good",
			Title:      title,
			Text:       resolvedText,
			MarkdownIn: []string{"text"},
		}
		if n.cfg.Footer != "" {
			footer, err := renderOrDefault(n.cfg.Footer, func() (string, error) { return "", nil })
			if err != nil {
				return fmt.Errorf("render resolved alert footer: %w", err)
			}
			att.Footer = footer
		}
		attachments = append(attachments, att)
	}

	if len(attachments) == 0 {
		return nil
	}

	_, _, err := n.cli.PostMessageContext(ctx,
		n.cfg.Channel,
		slack.MsgOptionAttachments(attachments...),
	)
	if err != nil {
		return err
	}
	n.logger.Info("slack notification sent",
		logger.String("channel", n.cfg.Channel),
		logger.Int32("attachments", int32(len(attachments))))
	return nil
}

// slackAttachmentResult captures the rendered content of one Slack attachment.
type slackAttachmentResult struct {
	Color     string `json:"color"`
	Title     string `json:"title"`
	TitleLink string `json:"titleLink,omitempty"`
	Text      string `json:"text"`
	Footer    string `json:"footer,omitempty"`
	Ts        string `json:"ts,omitempty"`
}

// slackDryRunResult is the plugin-specific output returned by DryRun.
type slackDryRunResult struct {
	Channel     string                  `json:"channel"`
	Attachments []slackAttachmentResult `json:"attachments"`
}

// DryRun renders Slack attachments and optionally posts the message to the channel.
func (n *notifier) DryRun(ctx context.Context, data *template.NotificationData, send bool) (json.RawMessage, error) {
	// renderOrDefault renders the template string via engine (with org template blocks merged in)
	// when name is non-empty; falls back to defaultFn otherwise.
	renderOrDefault := func(name string, defaultFn func() (string, error)) (string, error) {
		if name != "" && n.engine != nil {
			return n.engine.Render(data.OrgID, name, data)
		}
		return defaultFn()
	}

	// renderer is used only inside the defaultFn lambdas for built-in title/body templates.
	renderer := template.NewRenderer()

	var results []slackAttachmentResult

	for _, alert := range data.FiringAlerts {
		alertData := &template.NotificationData{
			OrgID:             data.OrgID,
			Alerts:            []template.TemplateAlert{alert},
			FiringAlerts:      []template.TemplateAlert{alert},
			CommonLabels:      data.CommonLabels,
			CommonAnnotations: data.CommonAnnotations,
			GroupLabels:       data.GroupLabels,
			ReceiverName:      data.ReceiverName,
			Status:            "firing",
			ExternalURL:       data.ExternalURL,
			OrgTemplates:      data.OrgTemplates,
		}
		title, err := renderOrDefault(n.cfg.Title, func() (string, error) {
			return renderer.RenderDefaultTitle(alertData)
		})
		if err != nil {
			return nil, fmt.Errorf("render firing alert title: %w", err)
		}
		body, err := renderOrDefault(n.cfg.Body, func() (string, error) {
			return renderer.RenderDefaultBody(alertData)
		})
		if err != nil {
			return nil, fmt.Errorf("render firing alert body: %w", err)
		}

		var titleLink string
		if data.ExternalURL != "" && alert.GeneratorURL != "" {
			titleLink = data.ExternalURL + alert.GeneratorURL
		} else {
			titleLink = alert.GeneratorURL
		}

		att := slackAttachmentResult{
			Color:     "danger",
			Title:     title,
			TitleLink: titleLink,
			Text:      body,
		}
		if alert.StartsAt > 0 {
			att.Ts = fmt.Sprintf("%d", alert.StartsAt/1000)
		}
		if n.cfg.Footer != "" {
			footer, err := renderOrDefault(n.cfg.Footer, func() (string, error) { return "", nil })
			if err != nil {
				return nil, fmt.Errorf("render firing alert footer: %w", err)
			}
			att.Footer = footer
		}
		results = append(results, att)
	}

	for _, alert := range data.ResolvedAlerts {
		resolvedData := &template.NotificationData{
			OrgID:             data.OrgID,
			Alerts:            []template.TemplateAlert{alert},
			ResolvedAlerts:    []template.TemplateAlert{alert},
			CommonLabels:      data.CommonLabels,
			CommonAnnotations: data.CommonAnnotations,
			GroupLabels:       data.GroupLabels,
			ReceiverName:      data.ReceiverName,
			Status:            "resolved",
			ExternalURL:       data.ExternalURL,
			OrgTemplates:      data.OrgTemplates,
		}
		title, err := renderOrDefault(n.cfg.Title, func() (string, error) {
			return renderer.RenderDefaultTitle(resolvedData)
		})
		if err != nil {
			return nil, fmt.Errorf("render resolved alert title: %w", err)
		}

		var resolvedText string
		if alert.EndsAt > 0 {
			resolvedText = fmt.Sprintf("Alert resolved at %s",
				time.UnixMilli(alert.EndsAt).Format("2006-01-02 15:04:05"))
		} else {
			resolvedText = "Alert resolved"
		}

		att := slackAttachmentResult{
			Color: "good",
			Title: title,
			Text:  resolvedText,
		}
		if n.cfg.Footer != "" {
			footer, err := renderOrDefault(n.cfg.Footer, func() (string, error) { return "", nil })
			if err != nil {
				return nil, fmt.Errorf("render resolved alert footer: %w", err)
			}
			att.Footer = footer
		}
		results = append(results, att)
	}

	if send && len(results) > 0 {
		if err := n.Notify(ctx, data); err != nil {
			return nil, err
		}
	}

	dryRunResult := slackDryRunResult{
		Channel:     n.cfg.Channel,
		Attachments: results,
	}
	return encoding.JSONMarshal(dryRunResult), nil
}
