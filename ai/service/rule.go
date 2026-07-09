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

package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/accesscontrol"
	agentapi "github.com/lindb/linsight/ai/api"
	rulescomp "github.com/lindb/linsight/ai/components/rules"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

var ruleSrvLog = logger.GetLogger("agent", "rule-service")

// ErrGlobalRuleForbidden is returned when a non-super-admin attempts to create or
// modify a global rule (org_id=0). Only RoleLin may manage global rules.
var ErrGlobalRuleForbidden = errors.New("only super admin may manage global rules")

// ErrNoUser is returned when no signed user (or org) is present in the request context.
// The Authorize middleware normally guarantees one; this is defense-in-depth so a
// missing user surfaces as a clean error instead of a nil panic.
var ErrNoUser = errors.New("no signed user in context")

// ruleCacheRefresher is the minimal surface RuleService needs from the rule engine:
// trigger an immediate cache reload after a write so changes take effect without
// waiting for the periodic ticker. Kept as a local interface to avoid coupling the
// service to the full engine implementation.
type ruleCacheRefresher interface {
	RefreshCache(ctx context.Context) error
}

// RuleService manages AgentRule CRUD with two-tier (global + per-org) scoping.
type RuleService interface {
	// ListRules returns global rules plus the rules owned by the org in ctx.
	ListRules(ctx context.Context) ([]*agentmodel.AgentRule, error)
	// GetRuleByUID returns a single rule visible to the org in ctx (own or global).
	GetRuleByUID(ctx context.Context, uid string) (*agentmodel.AgentRule, error)
	// CreateRule inserts a new rule and returns its generated UID. Creating a global
	// rule requires super-admin (RoleLin).
	CreateRule(ctx context.Context, req *agentapi.RuleRequest) (string, error)
	// UpdateRule applies a partial update to an existing rule within the caller's scope.
	UpdateRule(ctx context.Context, uid string, req *agentapi.RuleRequest) error
	// DeleteRule removes a rule within the caller's scope.
	DeleteRule(ctx context.Context, uid string) error
}

type ruleService struct {
	db        dbpkg.DB
	refresher ruleCacheRefresher
}

// NewRuleService creates a RuleService backed by db. refresher may be nil (e.g. when
// the agent runtime is not wired), in which case writes rely on the periodic ticker.
func NewRuleService(db dbpkg.DB, refresher ruleCacheRefresher) RuleService {
	return &ruleService{db: db, refresher: refresher}
}

// ListRules returns global rules (org_id=0) plus rules owned by the caller's org,
// ordered by priority then id for deterministic display and injection order.
func (s *ruleService) ListRules(ctx context.Context) ([]*agentmodel.AgentRule, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var rules []*agentmodel.AgentRule
	if err := s.db.RawDB().
		Where("org_id = ? or org_id = ?", agentmodel.GlobalRuleOrgID, orgID).
		Order("priority asc, id asc").
		Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// GetRuleByUID returns a rule the caller may see: either their org's rule or a global one.
func (s *ruleService) GetRuleByUID(ctx context.Context, uid string) (*agentmodel.AgentRule, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var rule agentmodel.AgentRule
	if err := s.db.Get(&rule, "uid = ? and (org_id = ? or org_id = ?)",
		uid, agentmodel.GlobalRuleOrgID, orgID); err != nil {
		return nil, err
	}
	return &rule, nil
}

// CreateRule builds an AgentRule from req, enforcing the super-admin gate for global
// rules, then triggers an immediate cache refresh.
func (s *ruleService) CreateRule(ctx context.Context, req *agentapi.RuleRequest) (string, error) {
	if req.RuleName == "" || req.Content == "" {
		return "", fmt.Errorf("ruleName and content are required")
	}
	if err := rulescomp.ValidateContent(req.Content); err != nil {
		return "", fmt.Errorf("invalid rule content: %w", err)
	}
	user, err := requireUser(ctx)
	if err != nil {
		return "", err
	}
	if req.IsGlobal && !isSuperAdmin(user.Role) {
		return "", ErrGlobalRuleForbidden
	}

	rule := &agentmodel.AgentRule{
		RuleName:  req.RuleName,
		Content:   req.Content,
		IsGlobal:  req.IsGlobal,
		Priority:  req.Priority,
		IsEnabled: req.IsEnabled,
	}
	rule.UID = uuid.GenerateShortUUID()
	rule.OrgID = scopeOrgID(req.IsGlobal, user.Org.ID)
	rule.CreatedBy = user.User.ID
	rule.UpdatedBy = user.User.ID
	if err := s.db.Create(rule); err != nil {
		return "", err
	}
	s.refreshCache(ctx)
	return rule.UID, nil
}

// UpdateRule replaces the mutable fields of an existing rule (full replace, not a
// sparse patch). ruleName and content are required so an omitted field can never
// silently blank the rule body or name. Global rules may only be edited by super
// admins; per-org rules are scoped to the caller's org so cross-org edits are impossible.
func (s *ruleService) UpdateRule(ctx context.Context, uid string, req *agentapi.RuleRequest) error {
	if req.RuleName == "" || req.Content == "" {
		return fmt.Errorf("ruleName and content are required")
	}
	if err := rulescomp.ValidateContent(req.Content); err != nil {
		return fmt.Errorf("invalid rule content: %w", err)
	}
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}

	// Load the existing rule within the caller's visibility to determine its scope.
	existing, err := s.GetRuleByUID(ctx, uid)
	if err != nil {
		return err
	}
	if existing.OrgID == agentmodel.GlobalRuleOrgID && !isSuperAdmin(user.Role) {
		return ErrGlobalRuleForbidden
	}

	// Map-based Updates writes zero values too; that is intended here (full replace),
	// and the required-field checks above prevent blanking ruleName/content.
	updates := map[string]interface{}{
		"rule_name":  req.RuleName,
		"content":    req.Content,
		"priority":   req.Priority,
		"is_enabled": req.IsEnabled,
		"updated_by": user.User.ID,
	}
	if err := s.db.RawDB().Model(&agentmodel.AgentRule{}).
		Where("uid = ? and org_id = ?", uid, existing.OrgID).
		Updates(updates).Error; err != nil {
		return err
	}
	s.refreshCache(ctx)
	return nil
}

// DeleteRule removes a rule. Global-rule deletion requires super admin; per-org
// deletion is constrained to the caller's org.
func (s *ruleService) DeleteRule(ctx context.Context, uid string) error {
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}
	existing, err := s.GetRuleByUID(ctx, uid)
	if err != nil {
		return err
	}
	if existing.OrgID == agentmodel.GlobalRuleOrgID && !isSuperAdmin(user.Role) {
		return ErrGlobalRuleForbidden
	}
	if err := s.db.Delete(&agentmodel.AgentRule{}, "uid = ? and org_id = ?", uid, existing.OrgID); err != nil {
		return err
	}
	s.refreshCache(ctx)
	return nil
}

// refreshCache triggers an immediate cache reload when a refresher is wired. Failures
// are non-fatal: the periodic ticker will eventually reconcile the cache.
func (s *ruleService) refreshCache(ctx context.Context) {
	if s.refresher == nil {
		return
	}
	if err := s.refresher.RefreshCache(ctx); err != nil {
		ruleSrvLog.Warn("rules: post-write cache refresh failed", logger.Error(err))
	}
}

// requireUser extracts the signed user from ctx, returning ErrNoUser when the user
// or its org is absent so callers never dereference a nil pointer.
func requireUser(ctx context.Context) (*model.SignedUser, error) {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil || u.User == nil {
		return nil, ErrNoUser
	}
	return u, nil
}

// scopeOrgID maps the IsGlobal flag to the stored org_id.
func scopeOrgID(isGlobal bool, orgID int64) int64 {
	if isGlobal {
		return agentmodel.GlobalRuleOrgID
	}
	return orgID
}

// isSuperAdmin reports whether role is the super-admin role (RoleLin).
func isSuperAdmin(role accesscontrol.RoleType) bool {
	return role == accesscontrol.RoleLin
}
