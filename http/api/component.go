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

// ComponentAPI represents component information related api handlers.
type ComponentAPI struct {
	deps *depspkg.API
}

// NewComponentAPI creates a ComponentAPI instance.
func NewComponentAPI(deps *depspkg.API) *ComponentAPI {
	return &ComponentAPI{
		deps: deps,
	}
}

// LoadComponentTree returns component tree.
func (api *ComponentAPI) LoadComponentTree(c *gin.Context) {
	cmps, err := api.deps.CmpSrv.LoadComponentTree(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, cmps)
}

// CreateComponent creates a new component.
func (api *ComponentAPI) CreateComponent(c *gin.Context) {
	cmp := &model.Component{}
	if err := c.ShouldBind(cmp); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.CmpSrv.CreateComponent(c.Request.Context(), cmp)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateComponent updates component.
func (api *ComponentAPI) UpdateComponent(c *gin.Context) {
	cmp := &model.Component{}
	if err := c.ShouldBind(cmp); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.CmpSrv.UpdateComponent(c.Request.Context(), cmp)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Component updated")
}

// DeleteComponentByUID deletes component.
func (api *ComponentAPI) DeleteComponentByUID(c *gin.Context) {
	err := api.deps.CmpSrv.DeleteComponentByUID(c.Request.Context(), c.Param(constant.UID))
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Component deleted")
}

// SortComponents sorts component list.
func (api *ComponentAPI) SortComponents(c *gin.Context) {
	cmps := model.Components{}
	if err := c.ShouldBind(&cmps); err != nil {
		httppkg.Error(c, err)
		return
	}
	err := api.deps.CmpSrv.SortComponents(c.Request.Context(), cmps)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Components sorted")
}
