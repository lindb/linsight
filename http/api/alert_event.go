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

// AlertEventAPI handles HTTP requests related to alert events.
type AlertEventAPI struct {
	deps *depspkg.API
}

// NewAlertEventAPI creates an AlertEventAPI instance.
func NewAlertEventAPI(deps *depspkg.API) *AlertEventAPI {
	return &AlertEventAPI{deps: deps}
}

// SearchAlertEvents queries alert events with optional filters.
func (api *AlertEventAPI) SearchAlertEvents(c *gin.Context) {
	req := &model.SearchAlertEventRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	events, total, err := api.deps.AlertEventSrv.SearchAlertEvents(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":  total,
		"events": events,
	})
}

// GetAlertEventByUID returns a single alert event by UID.
func (api *AlertEventAPI) GetAlertEventByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	event, err := api.deps.AlertEventSrv.GetAlertEventByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, event)
}
