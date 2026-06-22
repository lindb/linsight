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

package webhook

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

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
			Type:         "Webhook",
			DisplayName:  "Webhook",
			Description:  "Send alert notifications to a custom HTTP endpoint",
			ConfigSchema: json.RawMessage(schema),
		},
		NewFn:      newWebhookNotifier,
		SummaryKey: "url", // show destination URL in the contact-point overview
	})
}

// Config holds per-receiver webhook configuration.
type Config struct {
	URL        string            `json:"url"`        // destination URL
	Method     string            `json:"method"`     // HTTP method (default POST)
	Headers    map[string]string `json:"headers"`    // optional extra headers
	AuthHeader string            `json:"authHeader"` // optional Authorization header value
	// Optional named template name ({{define "NAME"}} block from org templates) for the request body.
	// Falls back to JSON-serialized NotificationData when empty or OrgTemplates is absent.
	Content string `json:"content"`
}

type webhookNotifier struct {
	cfg    Config
	engine *template.RenderingEngine
	client *http.Client
	logger logger.Logger
}

func newWebhookNotifier(settingJSON, cfgJSON json.RawMessage, engine *template.RenderingEngine) (types.Notifier, error) {
	cRaw, _ := cfgJSON.MarshalJSON()
	var cfgData Config
	if err := encoding.JSONUnmarshal(cRaw, &cfgData); err != nil {
		return nil, err
	}
	if cfgData.Method == "" {
		cfgData.Method = http.MethodPost
	}
	return &webhookNotifier{
		cfg:    cfgData,
		engine: engine,
		client: &http.Client{},
		logger: logger.GetLogger("Alerting", "WebhookNotifier"),
	}, nil
}

func (n *webhookNotifier) Notify(ctx context.Context, data *template.NotificationData) error {
	if n.cfg.URL == "" {
		return fmt.Errorf("webhook: URL not configured")
	}

	// Use content template when configured; fall back to JSON-serialized data.
	var payload []byte
	if n.cfg.Content != "" && n.engine != nil {
		rendered, err := n.engine.Render(data.OrgID, n.cfg.Content, data)
		if err != nil {
			return fmt.Errorf("webhook: render content template: %w", err)
		}
		payload = []byte(rendered)
	} else {
		payload = encoding.JSONMarshal(data)
	}

	n.logger.Info("sending webhook notification",
		logger.String("url", n.cfg.URL),
		logger.String("method", n.cfg.Method),
		logger.String("payload", string(payload)))

	req, err := http.NewRequestWithContext(ctx, n.cfg.Method, n.cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if n.cfg.AuthHeader != "" {
		req.Header.Set("Authorization", n.cfg.AuthHeader)
	}
	for k, v := range n.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: unexpected status %d", resp.StatusCode)
	}
	n.logger.Info("webhook notification sent",
		logger.String("url", n.cfg.URL),
		logger.Int32("status", int32(resp.StatusCode)))
	return nil
}

// webhookDryRunResult is the plugin-specific output returned by DryRun.
type webhookDryRunResult struct {
	URL     string `json:"url"`
	Method  string `json:"method"`
	Payload string `json:"payload"`
}

// DryRun renders the webhook payload and optionally dispatches it.
func (n *webhookNotifier) DryRun(ctx context.Context, data *template.NotificationData, send bool) (json.RawMessage, error) {
	if n.cfg.URL == "" {
		return nil, fmt.Errorf("webhook: URL not configured")
	}

	var payload []byte
	if n.cfg.Content != "" && n.engine != nil {
		rendered, err := n.engine.Render(data.OrgID, n.cfg.Content, data)
		if err != nil {
			return nil, fmt.Errorf("webhook: render content template: %w", err)
		}
		payload = []byte(rendered)
	} else {
		payload = encoding.JSONMarshal(data)
	}

	if send {
		if err := n.Notify(ctx, data); err != nil {
			return nil, err
		}
	}

	result := webhookDryRunResult{
		URL:     n.cfg.URL,
		Method:  n.cfg.Method,
		Payload: string(payload),
	}
	return encoding.JSONMarshal(result), nil
}
