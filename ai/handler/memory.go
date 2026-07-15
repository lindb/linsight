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

// MemoryHandler handles Memory CRUD HTTP requests under /ai/settings/memories.
type MemoryHandler struct {
	deps *Deps
}

// NewMemoryHandler creates a MemoryHandler wired to the given dependencies.
func NewMemoryHandler(deps *Deps) *MemoryHandler {
	return &MemoryHandler{deps: deps}
}

// ListMemories handles GET /ai/settings/memories.
func (h *MemoryHandler) ListMemories(c *gin.Context) {
	memories, err := h.deps.MemorySrv.ListMemories(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"memories": memories, "total": len(memories)})
}

// GetMemory handles GET /ai/settings/memories/:uid.
func (h *MemoryHandler) GetMemory(c *gin.Context) {
	uid := c.Param(constant.UID)
	memory, err := h.deps.MemorySrv.GetMemoryByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, memory)
}

// CreateMemory handles POST /ai/settings/memories. Returns the generated UID.
func (h *MemoryHandler) CreateMemory(c *gin.Context) {
	var req agentapi.MemoryRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := h.deps.MemorySrv.CreateMemory(c.Request.Context(), &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"uid": uid})
}

// UpdateMemory handles PUT /ai/settings/memories/:uid.
func (h *MemoryHandler) UpdateMemory(c *gin.Context) {
	var req agentapi.MemoryRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid := c.Param(constant.UID)
	if err := h.deps.MemorySrv.UpdateMemory(c.Request.Context(), uid, &req); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// DeleteMemory handles DELETE /ai/settings/memories/:uid.
func (h *MemoryHandler) DeleteMemory(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.MemorySrv.DeleteMemory(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
