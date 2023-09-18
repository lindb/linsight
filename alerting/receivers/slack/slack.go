package slack

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/slack-go/slack"

	"github.com/lindb/common/pkg/encoding"

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

type Setting struct {
	Token string `json:"token"`
}
type Config struct {
	Channel string
}

type notifer struct {
	setting Setting
	cfg     Config

	cli *slack.Client
}

func newSlackNotifer(setting, cfg json.RawMessage) (types.Notifier, error) {
	settingJSON, _ := setting.MarshalJSON()
	var settingData Setting
	if err := encoding.JSONUnmarshal(settingJSON, &settingData); err != nil {
		return nil, err
	}
	cfgJSON, _ := cfg.MarshalJSON()
	var cfgData Config
	if err := encoding.JSONUnmarshal(cfgJSON, &cfgData); err != nil {
		return nil, err
	}
	return &notifer{
		setting: settingData,
		cfg:     cfgData,
		cli:     slack.New(settingData.Token),
	}, nil
}

func (n *notifer) Notify(ctx context.Context, alerts []*model.Alert) error {
	tmplStr := `Metric:{{.MetricName}}
Field:{{.Field}}
Value:{{.Value}}
Tags:
{{- range $key, $value := .Tags }}
{{ $key }}:{{ $value }}
{{- end }}`
	var attachments []slack.Attachment
	for _, alert := range alerts {
		tmpl, err := template.New("fruits").Parse(tmplStr)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, alert.Series)
		if err != nil {
			panic(err)
		}
		msg := buf.String()
		fmt.Println(msg)
		attachments = append(attachments,
			slack.Attachment{
				Color:      "danger",
				Title:      "Memory Usage > 90%",
				TitleLink:  "http://localhost:3000/explore?left=%7B%22targets%22%3A%5B%7B%22refId%22%3A%22A%22%2C%22request%22%3A%7B%22namespace%22%3A%22default-ns%22%2C%22metric%22%3A%22lindb.monitor.system.mem_stat%22%2C%22fields%22%3A%5B%22usage%22%5D%2C%22groupBy%22%3A%5B%22node%22%5D%7D%7D%5D%2C%22type%22%3A%22timeseries%22%2C%22fieldConfig%22%3A%7B%22defaults%22%3A%7B%22unit%22%3A%22short%22%7D%7D%2C%22datasource%22%3A%7B%22uid%22%3A%22YJi4skUVz%22%2C%22type%22%3A%22lindb%22%7D%7D",
				Text:       msg,
				MarkdownIn: []string{"text"},
				Ts:         json.Number(fmt.Sprintf("%d", alert.Series.Timestamp)),
			},
		)
	}
	_, _, err := n.cli.PostMessage(
		n.cfg.Channel,
		slack.MsgOptionAttachments(attachments...),
	)
	if err != nil {
		return err
	}
	return nil
}
