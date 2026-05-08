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

package mail

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

//go:embed setting.json
var setting string

func init() {
	types.Register(&types.ReceiverPluginDefine{
		Define: &model.ReceiverPlugin{
			Type:    "Email",
			Setting: json.RawMessage(setting),
		},
		NewFn: newEmailNotifier,
	})
}

// Setting holds global SMTP server configuration stored in NotificationSetting.
type Setting struct {
	Host     string `json:"host"`     // SMTP server host
	Port     int    `json:"port"`     // SMTP server port (e.g. 587)
	Username string `json:"username"` // SMTP auth username
	Password string `json:"password"` // SMTP auth password
	From     string `json:"from"`     // sender email address
	StartTLS bool   `json:"startTLS"` // use STARTTLS when true
}

// Config holds per-receiver email configuration.
type Config struct {
	Addresses string `json:"addresses"` // comma or newline separated recipient addresses
}

type emailNotifier struct {
	setting Setting
	cfg     Config
}

func newEmailNotifier(settingJSON, cfgJSON json.RawMessage) (types.Notifier, error) {
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
	return &emailNotifier{
		setting: settingData,
		cfg:     cfgData,
	}, nil
}

func (n *emailNotifier) Notify(ctx context.Context, data *template.NotificationData) error {
	if n.cfg.Addresses == "" {
		return fmt.Errorf("email: no recipient addresses configured")
	}
	if n.setting.Host == "" {
		return fmt.Errorf("email: SMTP host not configured")
	}

	renderer := template.NewRenderer()

	// render subject and body
	subject, err := renderer.RenderDefaultTitle(data)
	if err != nil {
		subject = fmt.Sprintf("[%s] Alert notification", strings.ToUpper(data.Status))
	}

	body, err := renderer.RenderDefaultBody(data)
	if err != nil {
		body = fmt.Sprintf("Alert notification for %s", data.ReceiverName)
	}

	recipients := parseAddresses(n.cfg.Addresses)
	if len(recipients) == 0 {
		return fmt.Errorf("email: no valid recipients")
	}

	// build raw message
	msg := buildMessage(n.setting.From, recipients, subject, body)

	addr := fmt.Sprintf("%s:%d", n.setting.Host, n.setting.Port)

	// check context before blocking on network
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if n.setting.StartTLS {
		return n.sendWithTLS(addr, msg, recipients)
	}
	return n.sendPlain(addr, msg, recipients)
}

func (n *emailNotifier) sendPlain(addr string, msg []byte, recipients []string) error {
	var auth smtp.Auth
	if n.setting.Username != "" {
		host := strings.Split(addr, ":")[0]
		auth = smtp.PlainAuth("", n.setting.Username, n.setting.Password, host)
	}
	return smtp.SendMail(addr, auth, n.setting.From, recipients, msg)
}

func (n *emailNotifier) sendWithTLS(addr string, msg []byte, recipients []string) error {
	host := strings.Split(addr, ":")[0]
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("email TLS dial: %w", err)
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("email SMTP client: %w", err)
	}
	defer client.Close()

	if n.setting.Username != "" {
		auth := smtp.PlainAuth("", n.setting.Username, n.setting.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email auth: %w", err)
		}
	}

	if err := client.Mail(n.setting.From); err != nil {
		return fmt.Errorf("email MAIL FROM: %w", err)
	}
	for _, r := range recipients {
		if err := client.Rcpt(r); err != nil {
			return fmt.Errorf("email RCPT TO %s: %w", r, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("email DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("email write body: %w", err)
	}
	return w.Close()
}

// parseAddresses splits a comma- or newline-separated address list.
func parseAddresses(raw string) []string {
	raw = strings.NewReplacer("\n", ",", ";", ",").Replace(raw)
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// buildMessage constructs a minimal RFC 2822 message.
func buildMessage(from string, to []string, subject, body string) []byte {
	var sb strings.Builder
	sb.WriteString("From: " + from + "\r\n")
	sb.WriteString("To: " + strings.Join(to, ", ") + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	return []byte(sb.String())
}
