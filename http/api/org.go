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

	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

type OrgAPI struct {
	deps *depspkg.API
}

func NewOrgAPI(deps *depspkg.API) *OrgAPI {
	return &OrgAPI{
		deps: deps,
	}
}

func (api *OrgAPI) CreateOrg(c *gin.Context) {
	org := &model.Org{}
	if err := c.ShouldBind(org); err != nil {
		httppkg.Error(c, err)
		return
	}

	uid, err := api.deps.OrgSrv.CreateOrg(c.Request.Context(), org)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

func (api *OrgAPI) UpdateOrg(c *gin.Context) {
	org := &model.Org{}
	if err := c.ShouldBind(org); err != nil {
		httppkg.Error(c, err)
		return
	}

	if err := api.deps.OrgSrv.UpdateOrg(c.Request.Context(), org); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "update org success")
}

func (api *OrgAPI) DeleteOrg(c *gin.Context) {
	uid := c.Param("uid")
	if err := api.deps.OrgSrv.DeleteOrg(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "delete org success")
}

func (api *OrgAPI) GetOrg(c *gin.Context) {
	uid := c.Param("uid")
	org, err := api.deps.OrgSrv.GetOrgByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, org)
}

func (api *OrgAPI) SearchOrg(c *gin.Context) {
	orgs, err := api.deps.OrgSrv.SearchOrg(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, orgs)
}
