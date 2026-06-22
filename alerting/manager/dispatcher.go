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

// dispatcher resolves a receiver and sends notifications through all its enabled channels.
type dispatcher struct {
	deps            *Deps
	contactPointUID string
	orgID           int64
	logger          logger.Logger
}

func newDispatcher(deps *Deps, contactPointUID string, orgID int64) *dispatcher {
	return &dispatcher{
		deps:            deps,
		contactPointUID: contactPointUID,
		orgID:           orgID,
		logger:          logger.GetLogger("Alerting", "Dispatcher"),
	}
}

// dispatch builds notification data and sends it through all enabled channels of the receiver.
func (d *dispatcher) dispatch(events []*model.AlertEvent, groupLabels map[string]string) error {
	receiver, err := d.deps.ContactPointSrv.GetContactPointByOrgAndUID(d.orgID, d.contactPointUID)
	if err != nil {
		return fmt.Errorf("get contact point %s: %w", d.contactPointUID, err)
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

	// Populate OrgTemplates from the shared rendering engine (cached, avoids per-dispatch DB queries).
	data.OrgTemplates, _ = d.deps.RenderingEngine.GetMergedOrgTemplates(d.orgID)
	data.OrgID = d.orgID

	channels, err := receiver.ToChannels()
	if err != nil {
		return fmt.Errorf("decode receiver channels: %w", err)
	}

	d.logger.Info("dispatching notification",
		logger.String("receiver", receiver.Name),
		logger.String("status", status),
		logger.Int32("events", int32(len(events))),
		logger.Int32("channels", int32(len(channels))))

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}

		// send through the notifier
		if err := d.renderAndSend(ch, data); err != nil {
			d.logger.Error("send notification failure",
				logger.String("type", ch.Type),
				logger.Error(err))
			// continue with other channels even if one fails
		}
	}
	return nil
}

func (d *dispatcher) renderAndSend(ch model.NotificationChannel, data *template.NotificationData) error {
	pluginDef, ok := types.GetNotifierDef(ch.Type)
	if !ok {
		return fmt.Errorf("notifier plugin %q not registered", ch.Type)
	}

	// get the global setting for this channel type from NotificationSrv
	// (previously this incorrectly used NotificationTemplateSrv which serves template strings, not SMTP/Slack config)
	settings, err := d.deps.NotificationSrv.GetNotificationSettingsByOrg(d.orgID)
	if err != nil {
		settings = nil
	}

	var settingJSON json.RawMessage
	for _, s := range settings {
		if s.Type == ch.Type {
			data, e := s.Config.MarshalJSON()
			if e == nil {
				settingJSON = json.RawMessage(data)
			}
			break
		}
	}
	if len(settingJSON) == 0 {
		settingJSON = json.RawMessage("{}")
	}

	notifier, err := pluginDef.NewFn(settingJSON, ch.Config, d.deps.RenderingEngine)
	if err != nil {
		return fmt.Errorf("create notifier: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	notifyErr := notifier.Notify(ctx, data)
	durationMs := time.Since(start).Milliseconds()

	// Persist delivery outcome so the contact-point list can show health and last-sent info.
	deliveryStatus, errMsg := "ok", ""
	if notifyErr != nil {
		deliveryStatus, errMsg = "error", notifyErr.Error()
	}
	// Only record status when the channel has a stable UID (legacy channels without UIDs are skipped).
	if ch.UID != "" {
		_ = d.deps.ContactPointStatusSrv.UpsertChannelStatus(
			d.orgID, d.contactPointUID, ch.UID, ch.Type,
			deliveryStatus, errMsg,
			time.Now().UnixMilli(), durationMs,
		)
	}

	if notifyErr != nil {
		return notifyErr
	}
	d.logger.Info("notification sent",
		logger.String("type", ch.Type),
		logger.String("receiver", data.ReceiverName))
	return nil
}
