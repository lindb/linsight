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
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
)

// channelStatusVO is the per-channel view object returned by the contact-point list.
type channelStatusVO struct {
	Type       string `json:"type"`
	Enabled    bool   `json:"enabled"`
	Summary    string `json:"summary"`              // primary config value (url / addresses / channel)
	Status     string `json:"status"`               // "ok", "error", or "unknown" (never sent)
	LastSentAt int64  `json:"lastSentAt,omitempty"` // Unix milliseconds; omitted when never sent
	DurationMs int64  `json:"durationMs,omitempty"` // delivery round-trip; omitted when never sent
	ErrorMsg   string `json:"errorMsg,omitempty"`   // error detail; populated only when Status == "error"
}

// contactPointVO enriches a ContactPoint with per-channel health information.
type contactPointVO struct {
	model.ContactPoint
	ChannelStatus []channelStatusVO `json:"channelStatus"`
}

// extractSummary reads the SummaryKey value from a channel's JSON config.
// Returns an empty string when the key is absent or the config is invalid.
func extractSummary(cfgJSON json.RawMessage, summaryKey string) string {
	if summaryKey == "" || len(cfgJSON) == 0 {
		return ""
	}
	var m map[string]any
	if err := json.Unmarshal(cfgJSON, &m); err != nil {
		return ""
	}
	if v, ok := m[summaryKey]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ContactPointAPI handles HTTP requests for contact point management.
type ContactPointAPI struct {
	deps *depspkg.API
}

// NewContactPointAPI creates a ContactPointAPI instance.
func NewContactPointAPI(deps *depspkg.API) *ContactPointAPI {
	return &ContactPointAPI{deps: deps}
}

// CreateContactPoint creates a new contact point.
func (api *ContactPointAPI) CreateContactPoint(c *gin.Context) {
	cp := &model.ContactPoint{}
	if err := c.ShouldBind(cp); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.ContactPointSrv.CreateContactPoint(c.Request.Context(), cp)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateContactPoint updates an existing contact point.
func (api *ContactPointAPI) UpdateContactPoint(c *gin.Context) {
	cp := &model.ContactPoint{}
	if err := c.ShouldBind(cp); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.ContactPointSrv.UpdateContactPoint(c.Request.Context(), cp); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Contact point updated")
}

// DeleteContactPointByUID removes a contact point.
func (api *ContactPointAPI) DeleteContactPointByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.ContactPointSrv.DeleteContactPointByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Contact point deleted")
}

// GetContactPointByUID returns a contact point by UID.
func (api *ContactPointAPI) GetContactPointByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	cp, err := api.deps.ContactPointSrv.GetContactPointByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, cp)
}

// SearchContactPoints queries contact points with optional filters.
// Each contact point is enriched with per-channel summary and delivery health.
func (api *ContactPointAPI) SearchContactPoints(c *gin.Context) {
	req := &model.SearchContactPointRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	contactPoints, total, err := api.deps.ContactPointSrv.SearchContactPoints(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	// collect all contact-point UIDs for a single batch status query
	uids := make([]string, len(contactPoints))
	for i := range contactPoints {
		uids[i] = contactPoints[i].UID
	}
	orgID := util.GetUser(c.Request.Context()).Org.ID
	statusByUID, _ := api.deps.ContactPointStatusSrv.BatchGetByContactPoints(orgID, uids)

	// build a status lookup: cpUID → channelUID → ContactPointStatus
	type channelKey struct{ cpUID, chUID string }
	statusMap := make(map[channelKey]model.ContactPointStatus)
	for cpUID, statuses := range statusByUID {
		for _, s := range statuses {
			statusMap[channelKey{cpUID, s.ChannelUID}] = s
		}
	}

	vos := make([]contactPointVO, len(contactPoints))
	for i, cp := range contactPoints {
		channels, err := cp.ToChannels()
		if err != nil {
			// skip channel decode failures gracefully
			vos[i] = contactPointVO{ContactPoint: cp}
			continue
		}

		chStatuses := make([]channelStatusVO, 0, len(channels))
		for _, ch := range channels {
			pluginDef, ok := types.GetNotifierDef(ch.Type)

			vo := channelStatusVO{
				Type:    ch.Type,
				Enabled: ch.Enabled,
				Status:  "unknown",
			}
			if ok {
				vo.Summary = extractSummary(ch.Config, pluginDef.SummaryKey)
			}

			// populate delivery health if a record exists for this channel's UID
			if rec, found := statusMap[channelKey{cp.UID, ch.UID}]; found {
				vo.Status = rec.Status
				vo.LastSentAt = rec.LastSentAt
				vo.DurationMs = rec.DurationMs
				if rec.Status == "error" {
					vo.ErrorMsg = rec.ErrorMessage
				}
			}
			chStatuses = append(chStatuses, vo)
		}

		vos[i] = contactPointVO{
			ContactPoint:  cp,
			ChannelStatus: chStatuses,
		}
	}

	httppkg.OK(c, gin.H{
		"total":         total,
		"contactPoints": vos,
	})
}

// TestContactPoint sends a test notification through all enabled channels of a contact point.
// It reads the global notification setting for each channel type so the test uses real SMTP/Slack config.
func (api *ContactPointAPI) TestContactPoint(c *gin.Context) {
	req := &struct {
		ContactPointUID string `json:"contactPointUID"`
	}{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}

	cp, err := api.deps.ContactPointSrv.GetContactPointByUID(c.Request.Context(), req.ContactPointUID)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	// build a sample notification data payload
	testData := &template.NotificationData{
		Status:       "firing",
		ReceiverName: cp.Name,
		CommonLabels: map[string]string{"alertname": "TestAlert"},
		GroupLabels:  map[string]string{"alertname": "TestAlert"},
	}

	// load all global notification settings so each channel type has its real config
	settings, err := api.deps.NotificationSrv.GetAllNotificationSettings(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	// build a quick type → raw-JSON config map
	settingByType := make(map[string][]byte, len(settings))
	for i := range settings {
		data, err := settings[i].Config.MarshalJSON()
		if err == nil {
			settingByType[settings[i].Type] = data
		}
	}

	channels, err := cp.ToChannels()
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		pluginDef, ok := types.GetNotifierDef(ch.Type)
		if !ok {
			continue
		}
		settingJSON := settingByType[ch.Type]
		if len(settingJSON) == 0 {
			settingJSON = []byte("{}")
		}
		notifier, err := pluginDef.NewFn(settingJSON, ch.Config, api.deps.RenderingEngine)
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

// TestNotificationChannel sends a test notification for a single inline channel config
// without requiring the contact point to be saved first.
func (api *ContactPointAPI) TestNotificationChannel(c *gin.Context) {
	ch := &model.NotificationChannel{}
	if err := c.ShouldBind(ch); err != nil {
		httppkg.Error(c, err)
		return
	}

	pluginDef, ok := types.GetNotifierDef(ch.Type)
	if !ok {
		httppkg.Error(c, fmt.Errorf("unknown notifier type: %s", ch.Type))
		return
	}

	// load the global setting for this channel type
	settings, err := api.deps.NotificationSrv.GetAllNotificationSettings(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	var settingJSON []byte
	for i := range settings {
		if settings[i].Type == ch.Type {
			settingJSON, _ = settings[i].Config.MarshalJSON()
			break
		}
	}
	if len(settingJSON) == 0 {
		settingJSON = []byte("{}")
	}

	notifier, err := pluginDef.NewFn(settingJSON, ch.Config, api.deps.RenderingEngine)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	testData := &template.NotificationData{
		Status:       "firing",
		ReceiverName: "inline-test",
		CommonLabels: map[string]string{"alertname": "TestAlert"},
		GroupLabels:  map[string]string{"alertname": "TestAlert"},
	}
	if err := notifier.Notify(context.Background(), testData); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Test notification sent")
}

// GetNotifierPlugins returns all registered notifier plugin definitions.
func (api *ContactPointAPI) GetNotifierPlugins(c *gin.Context) {
	httppkg.OK(c, types.GetNotifierPlugins())
}
