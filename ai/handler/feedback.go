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

// FeedbackHandler handles message-feedback HTTP requests.
type FeedbackHandler struct {
	deps *Deps
}

// NewFeedbackHandler creates a FeedbackHandler wired to the given dependencies.
func NewFeedbackHandler(deps *Deps) *FeedbackHandler {
	return &FeedbackHandler{deps: deps}
}

// UpsertFeedback handles PUT /ai/conversations/:uid/messages/:messageUID/feedback.
// Creates a new feedback record or replaces an existing one for the message.
// Returns the persisted MessageFeedback record in the response body.
func (h *FeedbackHandler) UpsertFeedback(c *gin.Context) {
	convUID := c.Param(constant.UID)
	messageUID := c.Param("messageUID")

	var req agentapi.FeedbackRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}

	fb, err := h.deps.FeedbackSrv.UpsertFeedback(c.Request.Context(), convUID, messageUID, &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, fb)
}

// GetFeedback handles GET /ai/conversations/:uid/messages/:messageUID/feedback.
// Returns the existing feedback for the message, or 404 if none has been submitted.
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	convUID := c.Param(constant.UID)
	messageUID := c.Param("messageUID")

	fb, err := h.deps.FeedbackSrv.GetFeedback(c.Request.Context(), convUID, messageUID)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, fb)
}
