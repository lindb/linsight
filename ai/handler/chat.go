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
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
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

// Chat handles POST /agent/chat and POST /agent/conversations/:uid/chat.
// It upgrades the response to an SSE stream and forwards LLM output chunk-by-chunk.
// All errors are terminated by the runtime itself (finalizeError), which streams the
// terminal RUN_ERROR event; the handler only logs.
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

	if _, _, err := h.deps.Agent.Chat(c.Request.Context(), &req, writer, convUID); err != nil {
		// HTTP 200 + SSE headers are already committed. The runtime's finalizeError has
		// already streamed the terminal RUN_ERROR event, so we only log here.
		chatLog.Info("chat run ended with error", logger.Error(err))
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
// Returns the conversation history as a MESSAGES_SNAPSHOT AG-UI event so the
// frontend processes history with the same schema as live streaming events.
func (h *ChatHandler) GetConversationMessages(c *gin.Context) {
	uid := c.Param(constant.UID)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 {
		limit = 100
	}
	events, err := h.deps.ConvSrv.GetConversationEvents(c.Request.Context(), uid, limit)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	c.JSON(http.StatusOK, agui.Event{
		Type:   agui.MessagesSnapshot,
		Events: events,
	})
}
