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
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	aiutil "github.com/lindb/linsight/ai/util"
	"github.com/lindb/linsight/constant"
)

var chatLog = logger.GetLogger("agent", "handler")

// ChatHandler handles chat and conversation HTTP requests.
type ChatHandler struct {
	deps *Deps
}

// NewChatHandler creates a ChatHandler wired to the given dependencies.
func NewChatHandler(deps *Deps) *ChatHandler {
	return &ChatHandler{deps: deps}
}

// chatError translates a service-layer Chat error into an SSE error code and a user-facing message.
// It deliberately avoids exposing raw internal error strings (e.g. eino NodeRunError) to clients.
func chatError(err error) (agentapi.SSEErrorCode, string) {
	return aiutil.ClassifyError(err)
}

// Chat handles POST /agent/chat and POST /agent/conversations/:uid/chat.
// It upgrades the response to an SSE stream and forwards LLM output chunk-by-chunk.
// All service errors are translated by chatErrMsg and delivered as SSE error events.
func (h *ChatHandler) Chat(c *gin.Context) {
	var req agentapi.ChatRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	if len(req.Content) > h.deps.Cfg.MaxMessageLength {
		httppkg.Error(c, fmt.Errorf("message too long: max %d bytes", h.deps.Cfg.MaxMessageLength))
		return
	}

	// Extract optional conversation UID from URL (empty string for new conversations).
	convUID := c.Param(constant.UID)

	// Commit SSE headers before the first write so the client starts receiving events.
	writer := agentapi.NewSSEWriter(c)

	createdUID, err := h.deps.AgentSrv.Chat(c.Request.Context(), &req, writer, convUID)
	if err != nil {
		// HTTP 200 + SSE headers are already committed; deliver the error via the stream,
		// then send a done event so the client knows the stream has terminated cleanly.
		// createdUID is non-empty when the conversation was created before the error occurred,
		// allowing the frontend to update the URL and refetch the sidebar.
		code, msg := chatError(err)
		chatLog.Info("sending SSE error event", logger.String("code", string(code)), logger.String("msg", msg), logger.Error(err))
		if writeErr := writer.WriteError(code, msg); writeErr != nil {
			chatLog.Warn("failed to write SSE error event", logger.Error(writeErr))
		}
		if doneErr := writer.WriteDone(createdUID, ""); doneErr != nil {
			chatLog.Warn("failed to write SSE done event", logger.Error(doneErr))
		}
	}
}

// ListConversations handles GET /agent/conversations.
func (h *ChatHandler) ListConversations(c *gin.Context) {
	var req agentapi.SearchConversationsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	if req.Limit <= 0 {
		req.Limit = constant.Limit
	}

	convs, total, err := h.deps.ConvSrv.ListConversations(c.Request.Context(), req.Offset, req.Limit)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"total": total, "conversations": convs})
}

// DeleteConversation handles DELETE /agent/conversations/:uid.
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.ConvSrv.DeleteConversation(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// UpdateConversationTitle handles PUT /agent/conversations/:uid/title.
// Allows users to rename a conversation after it has been auto-titled.
func (h *ChatHandler) UpdateConversationTitle(c *gin.Context) {
	uid := c.Param(constant.UID)
	var req agentapi.UpdateConversationTitleRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	conv, err := h.deps.ConvSrv.GetConversation(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := h.deps.ConvSrv.UpdateConversationTitle(c.Request.Context(), conv.ID, req.Title); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// GetConversationMessages handles GET /agent/conversations/:uid/messages.
// Returns the conversation history for the session restore flow.
func (h *ChatHandler) GetConversationMessages(c *gin.Context) {
	uid := c.Param(constant.UID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	msgs, err := h.deps.ConvSrv.GetConversationMessages(c.Request.Context(), uid, limit)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"messages": msgs})
}
