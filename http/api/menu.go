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

// MenuAPI represents menu information related api handlers.
type MenuAPI struct {
	deps *depspkg.API
}

// NewMenuAPI creates a MenuAPI instance.
func NewMenuAPI(deps *depspkg.API) *MenuAPI {
	return &MenuAPI{
		deps: deps,
	}
}

// LoadMenuTree returns menu tree.
func (api *MenuAPI) LoadMenuTree(c *gin.Context) {
	menus, err := api.deps.MenuSrv.LoadMenuTree(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, menus)
}

// CreateMenu creates a new menu.
func (api *MenuAPI) CreateMenu(c *gin.Context) {
	menu := &model.Menu{}
	if err := c.ShouldBind(menu); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.MenuSrv.CreateMenu(c.Request.Context(), menu)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateMenu updates menu.
func (api *MenuAPI) UpdateMenu(c *gin.Context) {
	menu := &model.Menu{}
	if err := c.ShouldBind(menu); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.MenuSrv.UpdateMenu(c.Request.Context(), menu)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "menu updated")
}

// DeleteMenuByUID deletes menu.
func (api *MenuAPI) DeleteMenuByUID(c *gin.Context) {
	err := api.deps.MenuSrv.DeleteMenuByUID(c.Request.Context(), c.Param(constant.UID))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "menu deleted")
}

// SortMenus sorts menu list.
func (api *MenuAPI) SortMenus(c *gin.Context) {
	menus := model.Menus{}
	if err := c.ShouldBind(&menus); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.MenuSrv.SortMenus(c.Request.Context(), menus)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "menus sorted")
}

// SaveOrgMenus creates or removes menus for org's menus.
func (api *MenuAPI) SaveOrgMenus(c *gin.Context) {
	var menus []model.OrgMenuInfo
	if err := c.ShouldBind(&menus); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.MenuSrv.SaveOrgMenus(c.Request.Context(), c.Param(constant.UID), menus)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "org menus saved")
}

// UpdateRolesOfOrgMenus updates roles for current org.
func (api *MenuAPI) UpdateRolesOfOrgMenus(c *gin.Context) {
	var menus []model.OrgMenuInfo
	if err := c.ShouldBind(&menus); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.MenuSrv.UpdateRolesOfOrgMenus(c.Request.Context(), menus)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "role of org. menus updated")
}

// GetMenuTreeByCurrentOrg returns menu tree that current user can access of current org.
func (api *MenuAPI) GetMenuTreeByCurrentOrg(c *gin.Context) {
	menus, err := api.deps.MenuSrv.GetMenuTreeByCurrentOrg(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, menus)
}

// GetOrgMenus returns menu list for org.
func (api *MenuAPI) GetOrgMenus(c *gin.Context) {
	menus, err := api.deps.MenuSrv.GetOrgMenus(c.Request.Context(), c.Param(constant.UID))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, menus)
}
