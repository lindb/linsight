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

// AlertRouteAPI handles HTTP requests for alert routing tree management.
type AlertRouteAPI struct {
	deps *depspkg.API
}

// NewAlertRouteAPI creates an AlertRouteAPI instance.
func NewAlertRouteAPI(deps *depspkg.API) *AlertRouteAPI {
	return &AlertRouteAPI{deps: deps}
}

// GetAlertRouteTree returns all route nodes for the signed-in org.
func (api *AlertRouteAPI) GetAlertRouteTree(c *gin.Context) {
	routes, err := api.deps.AlertRouteSrv.GetAlertRouteTree(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, routes)
}

// SearchAlertRoutes searches routes by name.
func (api *AlertRouteAPI) SearchAlertRoutes(c *gin.Context) {
	req := &model.SearchAlertRouteRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	routes, total, err := api.deps.AlertRouteSrv.SearchAlertRoutes(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":  total,
		"routes": routes,
	})
}

// GetAlertRouteByUID returns a single route node.
func (api *AlertRouteAPI) GetAlertRouteByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	route, err := api.deps.AlertRouteSrv.GetAlertRouteByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, route)
}

// CreateAlertRoute creates a new route node.
func (api *AlertRouteAPI) CreateAlertRoute(c *gin.Context) {
	route := &model.AlertRoute{}
	if err := c.ShouldBind(route); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.AlertRouteSrv.CreateAlertRoute(c.Request.Context(), route)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateAlertRoute updates a route node.
func (api *AlertRouteAPI) UpdateAlertRoute(c *gin.Context) {
	route := &model.AlertRoute{}
	if err := c.ShouldBind(route); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.AlertRouteSrv.UpdateAlertRoute(c.Request.Context(), route); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert route updated")
}

// DeleteAlertRouteByUID removes a route node.
func (api *AlertRouteAPI) DeleteAlertRouteByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.AlertRouteSrv.DeleteAlertRouteByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert route deleted")
}
