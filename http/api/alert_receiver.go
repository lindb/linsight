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
	"context"

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// AlertReceiverAPI handles HTTP requests for alert receiver management.
type AlertReceiverAPI struct {
	deps *depspkg.API
}

// NewAlertReceiverAPI creates an AlertReceiverAPI instance.
func NewAlertReceiverAPI(deps *depspkg.API) *AlertReceiverAPI {
	return &AlertReceiverAPI{deps: deps}
}

// CreateAlertReceiver creates a new receiver.
func (api *AlertReceiverAPI) CreateAlertReceiver(c *gin.Context) {
	receiver := &model.AlertReceiver{}
	if err := c.ShouldBind(receiver); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.AlertReceiverSrv.CreateAlertReceiver(c.Request.Context(), receiver)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateAlertReceiver updates an existing receiver.
func (api *AlertReceiverAPI) UpdateAlertReceiver(c *gin.Context) {
	receiver := &model.AlertReceiver{}
	if err := c.ShouldBind(receiver); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.AlertReceiverSrv.UpdateAlertReceiver(c.Request.Context(), receiver); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert receiver updated")
}

// DeleteAlertReceiverByUID removes a receiver.
func (api *AlertReceiverAPI) DeleteAlertReceiverByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.AlertReceiverSrv.DeleteAlertReceiverByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert receiver deleted")
}

// GetAlertReceiverByUID returns a receiver by UID.
func (api *AlertReceiverAPI) GetAlertReceiverByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	receiver, err := api.deps.AlertReceiverSrv.GetAlertReceiverByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, receiver)
}

// SearchAlertReceivers queries receivers with optional filters.
func (api *AlertReceiverAPI) SearchAlertReceivers(c *gin.Context) {
	req := &model.SearchAlertReceiverRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	receivers, total, err := api.deps.AlertReceiverSrv.SearchAlertReceivers(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":     total,
		"receivers": receivers,
	})
}

// TestAlertReceiver sends a test notification through all enabled channels of a receiver.
func (api *AlertReceiverAPI) TestAlertReceiver(c *gin.Context) {
	req := &struct {
		ReceiverUID string `json:"receiverUID"`
	}{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}

	// build a sample notification data payload
	testData := &template.NotificationData{
		Status:       "firing",
		ReceiverName: "test",
		CommonLabels: map[string]string{"alertname": "TestAlert"},
		GroupLabels:  map[string]string{"alertname": "TestAlert"},
	}

	receiver, err := api.deps.AlertReceiverSrv.GetAlertReceiverByUID(c.Request.Context(), req.ReceiverUID)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	testData.ReceiverName = receiver.Name

	channels, err := receiver.ToChannels()
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		pluginDef, ok := types.GetReceiverPlugin(ch.Type)
		if !ok {
			continue
		}
		notifier, err := pluginDef.NewFn([]byte("{}"), ch.Config)
		if err != nil {
			httppkg.Error(c, err)
			return
		}
		if err := notifier.Notify(context.Background(), testData); err != nil {
			httppkg.Error(c, err)
			return
		}
	}
	httppkg.OK(c, "Test notification sent")
}
