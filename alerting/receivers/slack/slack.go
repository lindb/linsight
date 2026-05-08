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
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/slack-go/slack"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

//go:embed setting.json
var setting string

//go:embed config.json
var config string

func init() {
	types.Register(&types.ReceiverPluginDefine{
		Define: &model.ReceiverPlugin{
			Type:    "Slack",
			Setting: json.RawMessage(setting),
			Config:  json.RawMessage(config),
		},
		NewFn: newSlackNotifer,
	})
}

// Setting holds global Slack plugin configuration (bot token).
type Setting struct {
	Token string `json:"token"`
}

// Config holds per-channel Slack configuration.
type Config struct {
	Channel     string `json:"channel"`
	TemplateUID string `json:"templateUID"` // optional: override notification body template
}

type notifer struct {
	setting Setting
	cfg     Config
	cli     *slack.Client
}

func newSlackNotifer(settingJSON, cfgJSON json.RawMessage) (types.Notifier, error) {
	sRaw, _ := settingJSON.MarshalJSON()
	var settingData Setting
	if err := encoding.JSONUnmarshal(sRaw, &settingData); err != nil {
		return nil, err
	}
	cRaw, _ := cfgJSON.MarshalJSON()
	var cfgData Config
	if err := encoding.JSONUnmarshal(cRaw, &cfgData); err != nil {
		return nil, err
	}
	return &notifer{
		setting: settingData,
		cfg:     cfgData,
		cli:     slack.New(settingData.Token),
	}, nil
}

func (n *notifer) Notify(ctx context.Context, data *template.NotificationData) error {
	renderer := template.NewRenderer()

	var attachments []slack.Attachment
	for _, alert := range data.FiringAlerts {
		// render body using the default template
		body, err := renderer.RenderDefaultBody(&template.NotificationData{
			Alerts:            []*model.AlertEvent{alert},
			FiringAlerts:      []*model.AlertEvent{alert},
			CommonLabels:      data.CommonLabels,
			CommonAnnotations: data.CommonAnnotations,
			GroupLabels:       data.GroupLabels,
			ReceiverName:      data.ReceiverName,
			Status:            "firing",
			ExternalURL:       data.ExternalURL,
		})
		if err != nil {
			body = fmt.Sprintf("[%s] value=%.2f", alert.RuleTitle, alert.Value)
		}

		// render title
		title, err := renderer.RenderDefaultTitle(data)
		if err != nil {
			title = fmt.Sprintf("[FIRING] %s", alert.RuleTitle)
		}

		var titleLink string
		if data.ExternalURL != "" && alert.GeneratorURL != "" {
			titleLink = data.ExternalURL + alert.GeneratorURL
		} else {
			titleLink = alert.GeneratorURL
		}

		var buf bytes.Buffer
		buf.WriteString(body)

		attachments = append(attachments, slack.Attachment{
			Color:      "danger",
			Title:      title,
			TitleLink:  titleLink,
			Text:       buf.String(),
			MarkdownIn: []string{"text"},
			Ts:         json.Number(fmt.Sprintf("%d", alert.StartsAt/1000)),
		})
	}

	for _, alert := range data.ResolvedAlerts {
		title, _ := renderer.RenderDefaultTitle(data)
		if title == "" {
			title = fmt.Sprintf("[RESOLVED] %s", alert.RuleTitle)
		}
		attachments = append(attachments, slack.Attachment{
			Color:      "good",
			Title:      title,
			Text:       fmt.Sprintf("Alert resolved at %d", alert.EndsAt),
			MarkdownIn: []string{"text"},
		})
	}

	if len(attachments) == 0 {
		return nil
	}

	_, _, err := n.cli.PostMessageContext(ctx,
		n.cfg.Channel,
		slack.MsgOptionAttachments(attachments...),
	)
	return err
}
