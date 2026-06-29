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

package handler

import (
	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/constant"
)

// ModelHandler handles model CRUD and org routing-config HTTP requests.
type ModelHandler struct {
	deps *Deps
}

// NewModelHandler creates a ModelHandler wired to the given dependencies.
func NewModelHandler(deps *Deps) *ModelHandler {
	return &ModelHandler{deps: deps}
}

// ── Model CRUD ────────────────────────────────────────────────────────────────

// ListModels handles GET /agent/models.
func (h *ModelHandler) ListModels(c *gin.Context) {
	models, err := h.deps.ModelSrv.ListModels(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"models": models})
}

// CreateModel handles POST /agent/models.
// Returns the generated UID of the created model.
func (h *ModelHandler) CreateModel(c *gin.Context) {
	var req agentapi.ModelRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := h.deps.ModelSrv.CreateModel(c.Request.Context(), &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"uid": uid})
}

// GetDefaultModel handles GET /agent/models/default.
// Must be registered before GET /agent/models/:uid to take precedence.
func (h *ModelHandler) GetDefaultModel(c *gin.Context) {
	model, err := h.deps.ModelSrv.GetDefaultModel(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, model)
}

// GetModel handles GET /agent/models/:uid.
func (h *ModelHandler) GetModel(c *gin.Context) {
	uid := c.Param(constant.UID)
	model, err := h.deps.ModelSrv.GetModelByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, model)
}

// UpdateModel handles PUT /agent/models/:uid.
func (h *ModelHandler) UpdateModel(c *gin.Context) {
	var req agentapi.ModelRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid := c.Param(constant.UID)
	if err := h.deps.ModelSrv.UpdateModel(c.Request.Context(), uid, &req); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// DeleteModel handles DELETE /agent/models/:uid.
func (h *ModelHandler) DeleteModel(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.ModelSrv.DeleteModel(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// SetDefaultModel handles PUT /agent/models/:uid/default.
// Atomically clears is_default for all org models and sets it on the target model.
func (h *ModelHandler) SetDefaultModel(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.ModelSrv.SetDefaultModel(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// ── Routing config ────────────────────────────────────────────────────────────

// GetOrgModelConfig handles GET /agent/settings/model-routing.
func (h *ModelHandler) GetOrgModelConfig(c *gin.Context) {
	cfg, err := h.deps.ModelSrv.GetOrgModelConfig(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, cfg)
}

// SaveOrgModelConfig handles PUT /agent/settings/model-routing.
func (h *ModelHandler) SaveOrgModelConfig(c *gin.Context) {
	var req agentapi.SaveOrgModelConfigRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := h.deps.ModelSrv.SaveOrgModelConfig(c.Request.Context(), &req); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
