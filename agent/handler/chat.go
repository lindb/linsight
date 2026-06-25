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
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	agentapi "github.com/lindb/linsight/agent/api"
	agentservice "github.com/lindb/linsight/agent/service"
	"github.com/lindb/linsight/constant"
)

// Chat handles POST /agent/chat and POST /agent/conversations/:uid/chat.
// It upgrades the response to an SSE stream and forwards LLM output chunk-by-chunk.
// ErrTooManyRequests is handled specially: SSE headers are already committed, so
// the error is delivered as an SSE error event rather than an HTTP status code.
func (h *AgentHandler) Chat(c *gin.Context) {
	var req agentapi.ChatRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	if len(req.Content) > h.cfg.MaxMessageLength {
		httppkg.Error(c, fmt.Errorf("message too long: max %d bytes", h.cfg.MaxMessageLength))
		return
	}

	// Extract optional conversation UID from URL (empty string for new conversations).
	convUID := c.Param(constant.UID)

	// Commit SSE headers before the first write so the client starts receiving events.
	writer := agentapi.NewSSEWriter(c)

	err := h.agentSrv.Chat(c.Request.Context(), &req, writer, convUID)
	if err == nil {
		return
	}
	if errors.Is(err, agentservice.ErrTooManyRequests) {
		// HTTP 200 + SSE headers are already sent; deliver the rejection via the stream.
		_ = writer.WriteError("too many concurrent requests", err)
		return
	}
	// For all other errors the service already wrote an SSE error event before returning.
}

// ListConversations handles GET /agent/conversations.
func (h *AgentHandler) ListConversations(c *gin.Context) {
	var req agentapi.SearchConversationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	if req.Limit <= 0 {
		req.Limit = constant.Limit
	}

	convs, total, err := h.settingsSrv.ListConversations(c.Request.Context(), req.Offset, req.Limit)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"total": total, "conversations": convs})
}

// DeleteConversation handles DELETE /agent/conversations/:uid.
func (h *AgentHandler) DeleteConversation(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.settingsSrv.DeleteConversation(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
