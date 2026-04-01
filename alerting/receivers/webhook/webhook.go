package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

func init() {
	types.Register(&types.ReceiverPluginDefine{
		Define: &model.ReceiverPlugin{
			Type:   "Webhook",
			Config: json.RawMessage(defaultConfig),
		},
		NewFn: newWebhookNotifier,
	})
}

// defaultConfig describes the webhook config form schema.
const defaultConfig = `[
  {"type":"Input","props":{"field":"url","label":"URL","rules":[{"required":true,"message":"URL is required"}]}},
  {"type":"Input","props":{"field":"method","label":"HTTP Method","default":"POST"}},
  {"type":"Input","props":{"field":"authHeader","label":"Authorization Header (optional)"}}
]`

// Config holds per-receiver webhook configuration.
type Config struct {
	URL        string            `json:"url"`        // destination URL
	Method     string            `json:"method"`     // HTTP method (default POST)
	Headers    map[string]string `json:"headers"`    // optional extra headers
	AuthHeader string            `json:"authHeader"` // optional Authorization header value
}

type webhookNotifier struct {
	cfg    Config
	client *http.Client
}

func newWebhookNotifier(settingJSON, cfgJSON json.RawMessage) (types.Notifier, error) {
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
		client: &http.Client{},
	}, nil
}

func (n *webhookNotifier) Notify(ctx context.Context, data *template.NotificationData) error {
	if n.cfg.URL == "" {
		return fmt.Errorf("webhook: URL not configured")
	}

	payload := encoding.JSONMarshal(data)

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
	return nil
}
