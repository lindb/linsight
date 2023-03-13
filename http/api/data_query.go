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

	apideps "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin/datasource"
)

type DataQueryAPI struct {
	deps *apideps.API
}

func NewDataQueryAPI(deps *apideps.API) *DataQueryAPI {
	return &DataQueryAPI{
		deps: deps,
	}
}

func (api *DataQueryAPI) Query(c *gin.Context) {
	req := &model.QueryRequest{}
	err := c.ShouldBind(&req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	ctx := c.Request.Context()
	rs := make([]any, len(req.Queries))
	for i, query := range req.Queries {
		ds, err := api.deps.DatasourceSrv.GetDatasourceByUID(ctx, query.Datasource.UID)
		if err != nil {
			httppkg.Error(c, err)
			return
		}
		// FIXME: check datasource plugin if exist
		dsRequest := datasource.DataSourceHandlers[ds.Type](ds.URL, json.RawMessage(ds.Config))
		resp, err := dsRequest.DataQuery(query, req.Range)
		if err != nil {
			httppkg.Error(c, err)
			return
		}
		rs[i] = resp
	}
	httppkg.OK(c, rs)
}
