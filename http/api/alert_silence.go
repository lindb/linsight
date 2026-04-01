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
	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// AlertSilenceAPI handles HTTP requests for alert silence management.
type AlertSilenceAPI struct {
	deps *depspkg.API
}

// NewAlertSilenceAPI creates an AlertSilenceAPI instance.
func NewAlertSilenceAPI(deps *depspkg.API) *AlertSilenceAPI {
	return &AlertSilenceAPI{deps: deps}
}

// CreateAlertSilence creates a new silence.
func (api *AlertSilenceAPI) CreateAlertSilence(c *gin.Context) {
	silence := &model.AlertSilence{}
	if err := c.ShouldBind(silence); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.AlertSilenceSrv.CreateAlertSilence(c.Request.Context(), silence)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateAlertSilence updates an existing silence.
func (api *AlertSilenceAPI) UpdateAlertSilence(c *gin.Context) {
	silence := &model.AlertSilence{}
	if err := c.ShouldBind(silence); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.AlertSilenceSrv.UpdateAlertSilence(c.Request.Context(), silence); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert silence updated")
}

// DeleteAlertSilenceByUID removes a silence.
func (api *AlertSilenceAPI) DeleteAlertSilenceByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.AlertSilenceSrv.DeleteAlertSilenceByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert silence deleted")
}

// GetAlertSilenceByUID returns a silence by UID.
func (api *AlertSilenceAPI) GetAlertSilenceByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	silence, err := api.deps.AlertSilenceSrv.GetAlertSilenceByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, silence)
}

// SearchAlertSilences queries silences.
func (api *AlertSilenceAPI) SearchAlertSilences(c *gin.Context) {
	req := &model.SearchAlertSilenceRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	silences, total, err := api.deps.AlertSilenceSrv.SearchAlertSilences(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":   total,
		"silences": silences,
	})
}
