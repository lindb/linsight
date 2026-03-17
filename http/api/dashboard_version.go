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
	"strconv"

	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	apideps "github.com/lindb/linsight/http/deps"
)

// DashboardVersionAPI represents dashboard version related api handlers.
type DashboardVersionAPI struct {
	deps apideps.API
}

// NewDashboardVersionAPI creates a DashboardVersionAPI instance.
func NewDashboardVersionAPI(deps *apideps.API) *DashboardVersionAPI {
	return &DashboardVersionAPI{
		deps: *deps,
	}
}

// GetDashboardVersions returns all dashboard versions.
func (api *DashboardVersionAPI) GetDashboardVersions(c *gin.Context) {
	uid := c.Param(constant.UID)
	ds, err := api.deps.DashboardVersionSrv.ListDashboardVersion(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ds)
}

// GetDashboardVersion returns dashboard version by uid and version.
func (api *DashboardVersionAPI) GetDashboardVersion(c *gin.Context) {
	uid := c.Param(constant.UID)
	version := c.Param("version")
	v, _ := strconv.ParseInt(version, 10, 32)
	ds, err := api.deps.DashboardVersionSrv.GetDashboardVersion(c.Request.Context(), uid, int(v))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ds)
}
