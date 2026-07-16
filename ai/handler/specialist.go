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

// SpecialistHandler handles Specialist (sub-agent) CRUD HTTP requests under
// /ai/settings/agents. The user-facing resource is "agents"; the internal type is
// Specialist (to avoid colliding with the agent runtime's Agent interface).
type SpecialistHandler struct {
	deps *Deps
}

// NewSpecialistHandler creates a SpecialistHandler wired to the given dependencies.
func NewSpecialistHandler(deps *Deps) *SpecialistHandler {
	return &SpecialistHandler{deps: deps}
}

// ListSpecialists handles GET /ai/settings/agents.
func (h *SpecialistHandler) ListSpecialists(c *gin.Context) {
	specialists, err := h.deps.SpecialistSrv.ListSpecialists(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"agents": specialists, "total": len(specialists)})
}

// GetSpecialist handles GET /ai/settings/agents/:uid.
func (h *SpecialistHandler) GetSpecialist(c *gin.Context) {
	uid := c.Param(constant.UID)
	specialist, err := h.deps.SpecialistSrv.GetSpecialistByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, specialist)
}

// CreateSpecialist handles POST /ai/settings/agents. Returns the generated UID.
func (h *SpecialistHandler) CreateSpecialist(c *gin.Context) {
	var req agentapi.AgentRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := h.deps.SpecialistSrv.CreateSpecialist(c.Request.Context(), &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"uid": uid})
}

// UpdateSpecialist handles PUT /ai/settings/agents/:uid.
func (h *SpecialistHandler) UpdateSpecialist(c *gin.Context) {
	var req agentapi.AgentRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid := c.Param(constant.UID)
	if err := h.deps.SpecialistSrv.UpdateSpecialist(c.Request.Context(), uid, &req); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// DeleteSpecialist handles DELETE /ai/settings/agents/:uid.
func (h *SpecialistHandler) DeleteSpecialist(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.SpecialistSrv.DeleteSpecialist(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
