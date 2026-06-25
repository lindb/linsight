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

	agentapi "github.com/lindb/linsight/agent/api"
	"github.com/lindb/linsight/constant"
)

// ListMCPServers handles GET /agent/settings/mcp.
func (h *AgentHandler) ListMCPServers(c *gin.Context) {
	servers, err := h.settingsSrv.ListMCPServers(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"mcps": servers})
}

// CreateMCPServer handles POST /agent/settings/mcp.
// Returns the generated UID of the created server.
func (h *AgentHandler) CreateMCPServer(c *gin.Context) {
	var req agentapi.CreateMCPServerRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := h.settingsSrv.CreateMCPServer(c.Request.Context(), &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"uid": uid})
}

// DeleteMCPServer handles DELETE /agent/settings/mcp/:uid.
func (h *AgentHandler) DeleteMCPServer(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.settingsSrv.DeleteMCPServer(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
