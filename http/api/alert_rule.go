package api

import (
	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
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
	rule := &model.AlertRule{}
	if err := c.ShouldBind(&rule); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	uid, err := api.deps.AlertRuleSrv.CreateAlertRule(ctx, rule)
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
		"total": total,
		"rules": rules,
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
