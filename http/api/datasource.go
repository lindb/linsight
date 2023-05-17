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
	apideps "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// DatasourceAPI represents data source related api handlers.
type DatasourceAPI struct {
	deps *apideps.API
}

// NewDatasourceAPI creates a DatasourceAPI instance.
func NewDatasourceAPI(deps *apideps.API) *DatasourceAPI {
	return &DatasourceAPI{
		deps: deps,
	}
}

// CreateDatasource creates a data source.
func (api *DatasourceAPI) CreateDatasource(c *gin.Context) {
	ds := &model.Datasource{}
	if err := c.ShouldBind(ds); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.DatasourceSrv.CreateDatasource(c.Request.Context(), ds)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateDatasource updates data source by uid.
func (api *DatasourceAPI) UpdateDatasource(c *gin.Context) {
	ds := &model.Datasource{}
	if err := c.ShouldBind(ds); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.DatasourceSrv.UpdateDatasource(c.Request.Context(), ds); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Data source updated")
}

// DeleteDatasource deletes data source by uid.
func (api *DatasourceAPI) DeleteDatasourceByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.DatasourceSrv.DeleteDatasourceByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Data source deleted")
}

// GetDatasource returns data source by uid.
func (api *DatasourceAPI) GetDatasourceByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	ds, err := api.deps.DatasourceSrv.GetDatasourceByUID(c.Request.Context(), uid)
	if err != nil {
		//TODO: check not found???
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ds)
}

// GetDatasources returns all data sources.
func (api *DatasourceAPI) GetDatasources(c *gin.Context) {
	dataSources, err := api.deps.DatasourceSrv.GetDatasources(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, dataSources)
}
