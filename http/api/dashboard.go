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
	"gorm.io/datatypes"

	"github.com/lindb/common/pkg/encoding"
	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
)

// DashboardAPI represents dashboard related api handlers.
type DashboardAPI struct {
	deps *depspkg.API
}

// NewDashboardAPI creates a DashboardAPI instance.
func NewDashboardAPI(deps *depspkg.API) *DashboardAPI {
	return &DashboardAPI{
		deps: deps,
	}
}

// CreateDashboard creates a new dashboard.
func (api *DashboardAPI) CreateDashboard(c *gin.Context) { //nolint:dupl
	var dashboardJSON datatypes.JSON
	if err := c.ShouldBind(&dashboardJSON); err != nil {
		httppkg.Error(c, err)
		return
	}
	dashboard := &model.Dashboard{
		Config: dashboardJSON,
	}
	dashboard.ReadMeta()
	ctx := c.Request.Context()
	uid, err := api.deps.DashboardSrv.CreateDashboard(ctx, dashboard)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateDashboard updates an existing dashboard.
func (api *DashboardAPI) UpdateDashboard(c *gin.Context) {
	var dashboardJSON datatypes.JSON
	if err := c.ShouldBind(&dashboardJSON); err != nil {
		httppkg.Error(c, err)
		return
	}
	dashboard := &model.Dashboard{
		Config: dashboardJSON,
	}
	dashboard.ReadMeta()
	ctx := c.Request.Context()
	if err := api.deps.DashboardSrv.UpdateDashboard(ctx, dashboard); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Dashboard updated")
}

// DeleteDashboardByUID deletes dashboard by given uid.
func (api *DashboardAPI) DeleteDashboardByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.DashboardSrv.DeleteDashboardByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Dashboard deleted")
}

// GetDashboardByUID gets dashboard by given uid.
func (api *DashboardAPI) GetDashboardByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	dashboard, err := api.deps.DashboardSrv.GetDashboardByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	var dashboardMap map[string]any
	if err := jsonUnmarshalFn(dashboard.Config, &dashboardMap); err != nil {
		httppkg.Error(c, err)
		return
	}
	dashboardMap[constant.UID] = uid
	httppkg.OK(c, gin.H{
		"dashboard": dashboardMap,
	})
}

// StarDashboard stars the dashboard by given uid.
func (api *DashboardAPI) StarDashboard(c *gin.Context) {
	uid := c.Param(constant.UID)
	err := api.deps.DashboardSrv.StarDashboard(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "star success")
}

// UnstarDashboard unstars the dashboard by given uid.
func (api *DashboardAPI) UnstarDashboard(c *gin.Context) {
	uid := c.Param(constant.UID)
	err := api.deps.DashboardSrv.UnstarDashboard(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "unstar success")
}

// SearchDashboards searches dashboards by given params.
func (api *DashboardAPI) SearchDashboards(c *gin.Context) {
	req := &model.SearchDashboardRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	dashboards, total, err := api.deps.DashboardSrv.SearchDashboards(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":      total,
		"dashboards": dashboards,
	})
}
