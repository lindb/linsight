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
	"gorm.io/datatypes"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// ChartAPI represents chart repo related api handlers.
type ChartAPI struct {
	deps *depspkg.API
}

// NewChartAPI creates a ChartAPI instance.
func NewChartAPI(deps *depspkg.API) *ChartAPI {
	return &ChartAPI{
		deps: deps,
	}
}

// CreateChart creates a new chart.
func (api *ChartAPI) CreateChart(c *gin.Context) { //nolint:dupl
	var chartJSON datatypes.JSON
	if err := c.ShouldBind(&chartJSON); err != nil {
		httppkg.Error(c, err)
		return
	}
	chart := &model.Chart{
		Model: chartJSON,
	}
	chart.ReadMeta()
	ctx := c.Request.Context()
	uid, err := api.deps.ChartSrv.CreateChart(ctx, chart)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	if err0 := api.saveChartMeta(ctx, chart); err0 != nil {
		httppkg.Error(c, err0)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateChart updates a chart.
func (api *ChartAPI) UpdateChart(c *gin.Context) {
	var chartJSON datatypes.JSON
	if err := c.ShouldBind(&chartJSON); err != nil {
		httppkg.Error(c, err)
		return
	}
	chart := &model.Chart{
		Model: chartJSON,
	}
	chart.ReadMeta()
	ctx := c.Request.Context()
	err := api.deps.ChartSrv.UpdateChart(ctx, chart)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	if err0 := api.saveChartMeta(ctx, chart); err0 != nil {
		httppkg.Error(c, err0)
		return
	}
	httppkg.OK(c, "Chart updated")
}

// SearchCharts searches charts by given params.
func (api *ChartAPI) SearchCharts(c *gin.Context) {
	req := &model.SearchChartRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	charts, total, err := api.deps.ChartSrv.SearchCharts(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":  total,
		"charts": charts,
	})
}

// DeleteChartByUID deletes chart by given uid.
func (api *ChartAPI) DeleteChartByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.ChartSrv.DeleteChartByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	// FIXME: delete chart metadata
	httppkg.OK(c, "Chart deleted")
}

// GetChartByUID returns chart by given uid.
func (api *ChartAPI) GetChartByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	chart, err := api.deps.ChartSrv.GetChartByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"model": chart.Model,
	})
}

// GetDashboardsByChartUID returns dashboards by chart.
func (api *ChartAPI) GetDashboardsByChartUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	dashboards, err := api.deps.DashboardSrv.GetDashboardsByChartUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, dashboards)
}

// UnlinkChartFromDashboard unlinks chart from dashboard.
func (api *ChartAPI) UnlinkChartFromDashboard(c *gin.Context) {
	uid := c.Param(constant.UID)
	err := api.deps.ChartSrv.UnlinkChartFromDashboard(c.Request.Context(), uid, c.Param("dashboardUID"))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Unlinked chart from dashboard")
}

// saveChartMeta saves chart metadata.
func (api *ChartAPI) saveChartMeta(ctx context.Context, chart *model.Chart) error {
	// connect integration
	if chart.Integration != "" {
		if err := api.deps.IntegrationSrv.ConnectSource(ctx, chart.Integration, chart.UID, model.ChartResource); err != nil {
			return err
		}
	} else {
		if err := api.deps.IntegrationSrv.DisconnectSource(ctx, chart.UID, model.ChartResource); err != nil {
			return err
		}
	}
	return nil
}
