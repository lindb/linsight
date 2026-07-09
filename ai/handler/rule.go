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

// RuleHandler handles AgentRule CRUD HTTP requests under /ai/settings/rules.
type RuleHandler struct {
	deps *Deps
}

// NewRuleHandler creates a RuleHandler wired to the given dependencies.
func NewRuleHandler(deps *Deps) *RuleHandler {
	return &RuleHandler{deps: deps}
}

// ListRules handles GET /ai/settings/rules.
func (h *RuleHandler) ListRules(c *gin.Context) {
	rules, err := h.deps.RuleSrv.ListRules(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"rules": rules, "total": len(rules)})
}

// GetRule handles GET /ai/settings/rules/:uid.
func (h *RuleHandler) GetRule(c *gin.Context) {
	uid := c.Param(constant.UID)
	rule, err := h.deps.RuleSrv.GetRuleByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, rule)
}

// CreateRule handles POST /ai/settings/rules. Returns the generated UID.
func (h *RuleHandler) CreateRule(c *gin.Context) {
	var req agentapi.RuleRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := h.deps.RuleSrv.CreateRule(c.Request.Context(), &req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"uid": uid})
}

// UpdateRule handles PUT /ai/settings/rules/:uid.
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	var req agentapi.RuleRequest
	if err := c.ShouldBind(&req); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid := c.Param(constant.UID)
	if err := h.deps.RuleSrv.UpdateRule(c.Request.Context(), uid, &req); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}

// DeleteRule handles DELETE /ai/settings/rules/:uid.
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := h.deps.RuleSrv.DeleteRule(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nil)
}
