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
	"github.com/lindb/linsight/pkg/util"
)

// AlertRuleAPI represents alert rule related api handlers.
type AlertRuleAPI struct {
	deps *depspkg.API
}

// NewAlertRuleAPI creates an AlertRuleAPI instance.
func NewAlertRuleAPI(deps *depspkg.API) *AlertRuleAPI {
	return &AlertRuleAPI{
		deps: deps,
	}
}

// CreateAlertRule creates a new alert rule.
func (api *AlertRuleAPI) CreateAlertRule(c *gin.Context) { //nolint:dupl
	req := &model.CreateAlertRuleRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	uid, err := api.deps.AlertRuleSrv.CreateAlertRule(ctx, req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateAlertRule updates an alert rule.
func (api *AlertRuleAPI) UpdateAlertRule(c *gin.Context) {
	rule := &model.AlertRule{}
	if err := c.ShouldBind(&rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	err := api.deps.AlertRuleSrv.UpdateAlertRule(ctx, rule)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert rule updated")
}

// SearchAlertRules searches alert rules by given params.
func (api *AlertRuleAPI) SearchAlertRules(c *gin.Context) {
	req := &model.SearchAlertRuleRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	rules, total, err := api.deps.AlertRuleSrv.SearchAlertRules(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":  total,
		"groups": rules,
	})
}

// DeleteAlertRuleByUID deletes alert rule by given uid.
func (api *AlertRuleAPI) DeleteAlertRuleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.AlertRuleSrv.DeleteAlertRuleByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Alert rule deleted")
}

// GetAlertRuleByUID returns alert rule by given uid.
func (api *AlertRuleAPI) GetAlertRuleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	rule, err := api.deps.AlertRuleSrv.GetAlertRuleByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, rule)
}

// DryRunAlertRule executes the full evaluation pipeline for the given rule without
// persisting any state or delivering any notifications.
// The response always contains a structured result; pipeline errors are reported
// inside AlertRuleDryRunResult.Stage / Error rather than as HTTP errors.
// Pass ?send=true to also dispatch real notifications to the configured contact points.
func (api *AlertRuleAPI) DryRunAlertRule(c *gin.Context) {
	rule := &model.AlertRule{}
	if err := c.ShouldBind(rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	// inject org from authenticated context; the request body won't carry orgID
	if signedUser := util.GetUser(c.Request.Context()); signedUser != nil {
		rule.OrgID = signedUser.Org.ID
	}
	send := c.Query("send") == "true"
	result, err := api.deps.DryRunner.Run(c.Request.Context(), rule, send)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, result)
}
