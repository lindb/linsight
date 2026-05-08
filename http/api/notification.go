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

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

type NotificationAPI struct {
	deps *depspkg.API
}

func NewNotificationAPI(deps *depspkg.API) *NotificationAPI {
	return &NotificationAPI{
		deps: deps,
	}
}

func (api *NotificationAPI) Test(c *gin.Context) {
	receiver := &model.Recevier{}
	if err := c.ShouldBind(receiver); err != nil {
		httppkg.Error(c, err)
		return
	}
	receiverDefine, _ := types.GetReceiverPlugin(receiver.Type)
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
	notifier, err := receiverDefine.NewFn(json.RawMessage(setting.Config.String()), receiver.Config)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	notifier.Notify(c.Request.Context(), nil)

	httppkg.OK(c, "ok")
}

func (api *NotificationAPI) GetAllNotificationSettings(c *gin.Context) {
	settings, err := api.deps.NotificationSrv.GetAllNotificationSettings(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, settings)
}

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

func (api *NotificationAPI) DeleteNotificationSettting(c *gin.Context) {
	notificationType := c.Param("type")
	if err := api.deps.NotificationSrv.DeleteNotificationSetting(c.Request.Context(), notificationType); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification setting deleted")
}

func (api *NotificationAPI) CreateNotification(c *gin.Context) {
	var notification model.Notification
	if err := c.ShouldBind(&notification); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	uid, err := api.deps.NotificationSrv.CreateNotification(ctx, &notification)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

func (api *NotificationAPI) UpdateNotification(c *gin.Context) {
	var notification model.Notification
	if err := c.ShouldBind(&notification); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	err := api.deps.NotificationSrv.UpdateNotification(ctx, &notification)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification updated")
}

func (api *NotificationAPI) SearchNotifications(c *gin.Context) {
	req := &model.SearchNotificationRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	notifications, total, err := api.deps.NotificationSrv.FindNotifications(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":         total,
		"notifications": notifications,
	})
}

func (api *NotificationAPI) GetNotificationByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	notification, err := api.deps.NotificationSrv.GetNotificationByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, notification)
}
