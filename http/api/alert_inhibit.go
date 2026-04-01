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

// AlertInhibitAPI handles HTTP requests for alert inhibit rule management.
type AlertInhibitAPI struct {
	deps *depspkg.API
}

// NewAlertInhibitAPI creates an AlertInhibitAPI instance.
func NewAlertInhibitAPI(deps *depspkg.API) *AlertInhibitAPI {
	return &AlertInhibitAPI{deps: deps}
}

// CreateAlertInhibitRule creates a new inhibit rule.
func (api *AlertInhibitAPI) CreateAlertInhibitRule(c *gin.Context) {
	rule := &model.AlertInhibitRule{}
	if err := c.ShouldBind(rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.AlertInhibitRuleSrv.CreateAlertInhibitRule(c.Request.Context(), rule)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateAlertInhibitRule updates an existing inhibit rule.
func (api *AlertInhibitAPI) UpdateAlertInhibitRule(c *gin.Context) {
	rule := &model.AlertInhibitRule{}
	if err := c.ShouldBind(rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.AlertInhibitRuleSrv.UpdateAlertInhibitRule(c.Request.Context(), rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert inhibit rule updated")
}

// DeleteAlertInhibitRuleByUID removes an inhibit rule.
func (api *AlertInhibitAPI) DeleteAlertInhibitRuleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.AlertInhibitRuleSrv.DeleteAlertInhibitRuleByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert inhibit rule deleted")
}

// GetAlertInhibitRuleByUID returns an inhibit rule by UID.
func (api *AlertInhibitAPI) GetAlertInhibitRuleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	rule, err := api.deps.AlertInhibitRuleSrv.GetAlertInhibitRuleByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, rule)
}

// SearchAlertInhibitRules queries inhibit rules with optional filters.
func (api *AlertInhibitAPI) SearchAlertInhibitRules(c *gin.Context) {
	req := &model.SearchAlertInhibitRuleRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	rules, total, err := api.deps.AlertInhibitRuleSrv.SearchAlertInhibitRules(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total": total,
		"rules": rules,
	})
}
