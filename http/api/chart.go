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
func (api *ChartAPI) CreateChart(c *gin.Context) {
	var chart model.Chart
	if err := c.ShouldBind(&chart); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	uid, err := api.deps.ChartSrv.CreateChart(ctx, &chart)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateChart updates a chart.
func (api *ChartAPI) UpdateChart(c *gin.Context) {
	var chart model.Chart
	if err := c.ShouldBind(&chart); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	err := api.deps.ChartSrv.UpdateChart(ctx, &chart)
	if err != nil {
		httppkg.Error(c, err)
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
	httppkg.OK(c, "Chart deleted")
}
