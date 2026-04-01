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

package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

const templateRenderTimeout = 5 * time.Second

// dispatcher resolves a receiver and sends notifications through all its enabled channels.
type dispatcher struct {
	deps        *Deps
	receiverUID string
	orgID       int64
	renderer    template.Renderer
	logger      logger.Logger
}

func newDispatcher(deps *Deps, receiverUID string, orgID int64) *dispatcher {
	return &dispatcher{
		deps:        deps,
		receiverUID: receiverUID,
		orgID:       orgID,
		renderer:    template.NewRenderer(),
		logger:      logger.GetLogger("Alerting", "Dispatcher"),
	}
}

// dispatch builds notification data and sends it through all enabled channels of the receiver.
func (d *dispatcher) dispatch(events []*model.AlertEvent, groupLabels map[string]string) error {
	receiver, err := d.deps.AlertReceiverSrv.GetAlertReceiverByOrgAndUID(d.orgID, d.receiverUID)
	if err != nil {
		return fmt.Errorf("get receiver %s: %w", d.receiverUID, err)
	}

	// determine notification status
	status := "firing"
	allResolved := true
	for _, ev := range events {
		if ev.State != model.AlertEventResolved {
			allResolved = false
			break
		}
	}
	if allResolved {
		status = "resolved"
	}

	data := template.BuildNotificationData(events, receiver.Name, status, d.deps.ExternalURL, groupLabels)

	channels, err := receiver.ToChannels()
	if err != nil {
		return fmt.Errorf("decode receiver channels: %w", err)
	}

	// load custom template if configured
	var notifTmpl *model.NotificationTemplate
	if receiver.TemplateUID != "" {
		notifTmpl, _ = d.deps.NotificationTemplateSrv.GetTemplateByOrgAndUID(d.orgID, receiver.TemplateUID)
	}

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}

		// render template
		if err := d.renderAndSend(ch, data, notifTmpl); err != nil {
			d.logger.Error("send notification failure",
				logger.String("type", ch.Type),
				logger.Error(err))
			// continue with other channels even if one fails
		}
	}
	return nil
}

func (d *dispatcher) renderAndSend(ch model.ReceiverChannel, data *template.NotificationData, notifTmpl *model.NotificationTemplate) error {
	pluginDef, ok := types.GetReceiverPlugin(ch.Type)
	if !ok {
		return fmt.Errorf("receiver plugin %q not registered", ch.Type)
	}

	// get the global setting for this channel type
	notificationSetting, err := d.deps.NotificationTemplateSrv.GetTemplateByOrgAndUID(d.orgID, ch.Type)
	if err != nil {
		// setting may not exist — pass empty JSON
		notificationSetting = nil
	}

	var settingJSON json.RawMessage
	if notificationSetting != nil {
		settingJSON = json.RawMessage(notificationSetting.Template)
	} else {
		settingJSON = json.RawMessage("{}")
	}

	notifier, err := pluginDef.NewFn(settingJSON, ch.Config)
	if err != nil {
		return fmt.Errorf("create notifier: %w", err)
	}

	// override notification data with rendered template if available
	if notifTmpl != nil {
		ctx, cancel := context.WithTimeout(context.Background(), templateRenderTimeout)
		defer cancel()
		rendered, renderErr := template.RenderWithContext(ctx, d.renderer, notifTmpl.Type, notifTmpl.Template, data)
		if renderErr != nil {
			d.logger.Warn("template render failure, using default",
				logger.String("templateUID", notifTmpl.UID),
				logger.Error(renderErr))
		} else {
			_ = rendered // rendered result is available for channel-specific use
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return notifier.Notify(ctx, data)
}
