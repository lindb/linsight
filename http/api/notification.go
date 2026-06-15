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

package api

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// testReceiverRequest holds the minimal fields needed for a test notification send.
type testReceiverRequest struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

type NotificationAPI struct {
	deps *depspkg.API
}

func NewNotificationAPI(deps *depspkg.API) *NotificationAPI {
	return &NotificationAPI{
		deps: deps,
	}
}

// Test sends a test notification through the given receiver configuration.
func (api *NotificationAPI) Test(c *gin.Context) {
	receiver := &testReceiverRequest{}
	if err := c.ShouldBind(receiver); err != nil {
		httppkg.Error(c, err)
		return
	}
	notifierDef, ok := types.GetNotifierDef(receiver.Type)
	if !ok {
		httppkg.Error(c, fmt.Errorf("unknown notifier type: %s", receiver.Type))
		return
	}
	settings, err := api.deps.NotificationSrv.GetAllNotificationSettings(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	var setting model.NotificationSetting
	for idx := range settings {
		if settings[idx].Type == receiver.Type {
			setting = settings[idx]
			break
		}
	}
	notifier, err := notifierDef.NewFn(json.RawMessage(setting.Config.String()), receiver.Config)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	// build a minimal test payload so notifiers do not receive nil data
	testData := &template.NotificationData{
		Status:       "firing",
		ReceiverName: receiver.Type,
		CommonLabels: map[string]string{"alertname": "TestAlert"},
		GroupLabels:  map[string]string{"alertname": "TestAlert"},
	}
	if err := notifier.Notify(c.Request.Context(), testData); err != nil {
		httppkg.Error(c, err)
		return
	}

	httppkg.OK(c, "ok")
}

// GetAllNotificationSettings returns all notification settings for the current org.
func (api *NotificationAPI) GetAllNotificationSettings(c *gin.Context) {
	settings, err := api.deps.NotificationSrv.GetAllNotificationSettings(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, settings)
}

// SaveNotificationSetting creates or updates a notification setting.
func (api *NotificationAPI) SaveNotificationSetting(c *gin.Context) {
	setting := &model.NotificationSetting{}
	if err := c.ShouldBind(setting); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.NotificationSrv.SaveNotificationSetting(c.Request.Context(), setting); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification setting saved")
}

// DeleteNotificationSettting removes a notification setting by type.
func (api *NotificationAPI) DeleteNotificationSettting(c *gin.Context) {
	notificationType := c.Param("type")
	if err := api.deps.NotificationSrv.DeleteNotificationSetting(c.Request.Context(), notificationType); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification setting deleted")
}
