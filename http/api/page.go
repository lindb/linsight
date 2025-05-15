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

// PageAPI represents page information related api handlers.
type PageAPI struct {
	deps *depspkg.API
}

// NewPageAPI creates a PageAPI instance.
func NewPageAPI(deps *depspkg.API) *PageAPI {
	return &PageAPI{
		deps: deps,
	}
}

// LoadPageTree returns page tree.
func (api *PageAPI) LoadPageTree(c *gin.Context) {
	pages, err := api.deps.PageSrv.LoadPageTree(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, pages)
}

// CreatePage creates a new page.
func (api *PageAPI) CreatePage(c *gin.Context) {
	page := &model.Page{}
	if err := c.ShouldBind(page); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.PageSrv.CreatePage(c.Request.Context(), page)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdatePage updates page.
func (api *PageAPI) UpdatePage(c *gin.Context) {
	page := &model.Page{}
	if err := c.ShouldBind(page); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.PageSrv.UpdatePage(c.Request.Context(), page)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "page updated")
}

// DeletePageByUID deletes page.
func (api *PageAPI) DeletePageByUID(c *gin.Context) {
	err := api.deps.PageSrv.DeletePageByUID(c.Request.Context(), c.Param(constant.UID))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "page deleted")
}

// SortPages sorts page list.
func (api *PageAPI) SortPages(c *gin.Context) {
	pages := model.Pages{}
	if err := c.ShouldBind(&pages); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.PageSrv.SortPages(c.Request.Context(), pages)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "pages sorted")
}

// SaveOrgPages creates or removes pages for org's pages.
func (api *PageAPI) SaveOrgPages(c *gin.Context) {
	var pages []model.OrgPageInfo
	if err := c.ShouldBind(&pages); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.PageSrv.SaveOrgPages(c.Request.Context(), c.Param(constant.UID), pages)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "org pages saved")
}

// UpdateRolesOfOrgPages updates roles for current org.
func (api *PageAPI) UpdateRolesOfOrgPages(c *gin.Context) {
	var pages []model.OrgPageInfo
	if err := c.ShouldBind(&pages); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.PageSrv.UpdateRolesOfOrgPages(c.Request.Context(), pages)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "role of org. pages updated")
}

// GetPageTreeByCurrentOrg returns page tree that current user can access of current org.
func (api *PageAPI) GetPageTreeByCurrentOrg(c *gin.Context) {
	pages, err := api.deps.PageSrv.GetPageTreeByCurrentOrg(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, pages)
}

// GetOrgPages returns page list for org.
func (api *PageAPI) GetOrgPages(c *gin.Context) {
	pages, err := api.deps.PageSrv.GetOrgPages(c.Request.Context(), c.Param(constant.UID))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, pages)
}
