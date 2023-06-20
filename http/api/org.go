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

// OrgAPI represents org related api handlers.
type OrgAPI struct {
	deps *depspkg.API
}

// NewOrgAPI creates an OrgAPI instance.
func NewOrgAPI(deps *depspkg.API) *OrgAPI {
	return &OrgAPI{
		deps: deps,
	}
}

// CreateOrg creates a new org.
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

// UpdateOrg updates an org.
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
	httppkg.OK(c, "Organization updated")
}

// DeleteOrgByUID deletes the org by given uid.
func (api *OrgAPI) DeleteOrgByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.OrgSrv.DeleteOrgByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Organization deleted")
}

// GetOrgByUID returns the org by given uid.
func (api *OrgAPI) GetOrgByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	org, err := api.deps.OrgSrv.GetOrgByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, org)
}

// SearchOrg searches organizations by given params.
func (api *OrgAPI) SearchOrg(c *gin.Context) {
	req := &model.SearchOrgRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	org, total, err := api.deps.OrgSrv.SearchOrg(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":         total,
		"organizations": org,
	})
}

// GetOrgListForSignedUser returns all org for current signed user can manage.
func (api *OrgAPI) GetOrgListForSignedUser(c *gin.Context) {
	orgs, err := api.deps.OrgSrv.GetOrgListForSignedUser(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, orgs)
}

// GetUserListForSignedOrg returns the users for current signed org, filter(user name/name/email).
func (api *OrgAPI) GetUserListForSignedOrg(c *gin.Context) {
	prefix := c.Query("prefix")
	users, err := api.deps.OrgSrv.GetUserListForSignedOrg(c.Request.Context(), prefix)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, users)
}
