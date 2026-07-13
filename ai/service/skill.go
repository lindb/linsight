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
	"fmt"

	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/uuid"
)

var skillSrvLog = logger.GetLogger("agent", "skill-service")

// SkillService manages Skill CRUD with two-tier (global + per-org) scoping.
type SkillService interface {
	// ListSkills returns global skills plus the skills owned by the org in ctx.
	ListSkills(ctx context.Context) ([]*agentmodel.Skill, error)
	// GetSkillByUID returns a single skill visible to the org in ctx (own or global).
	GetSkillByUID(ctx context.Context, uid string) (*agentmodel.Skill, error)
	// CreateSkill inserts a new skill and returns its generated UID. Creating a global
	// skill requires super-admin (RoleLin).
	CreateSkill(ctx context.Context, req *agentapi.SkillRequest) (string, error)
	// UpdateSkill applies a full replace update to an existing skill within the caller's scope.
	UpdateSkill(ctx context.Context, uid string, req *agentapi.SkillRequest) error
	// DeleteSkill removes a skill within the caller's scope.
	DeleteSkill(ctx context.Context, uid string) error
}

type skillService struct {
	db dbpkg.DB
}

// NewSkillService creates a SkillService backed by db. The service does not interact
// with the skill cache — cache freshness is maintained by DBBackend's background ticker.
func NewSkillService(db dbpkg.DB) SkillService {
	return &skillService{db: db}
}

// ListSkills returns global skills (org_id=0) plus skills owned by the caller's org.
func (s *skillService) ListSkills(ctx context.Context) ([]*agentmodel.Skill, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var skills []*agentmodel.Skill
	if err := s.db.RawDB().
		Where("org_id = ? or org_id = ?", agentmodel.GlobalRuleOrgID, orgID).
		Order("name asc").
		Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

// GetSkillByUID returns a skill the caller may see: either their org's skill or a global one.
func (s *skillService) GetSkillByUID(ctx context.Context, uid string) (*agentmodel.Skill, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var skill agentmodel.Skill
	if err := s.db.Get(&skill, "uid = ? and (org_id = ? or org_id = ?)",
		uid, agentmodel.GlobalRuleOrgID, orgID); err != nil {
		return nil, err
	}
	return &skill, nil
}

// CreateSkill builds a Skill from req, enforcing the super-admin gate for global skills.
func (s *skillService) CreateSkill(ctx context.Context, req *agentapi.SkillRequest) (string, error) {
	if req.Name == "" || req.Content == "" {
		return "", fmt.Errorf("name and content are required")
	}
	user, err := requireUser(ctx)
	if err != nil {
		return "", err
	}
	if req.IsGlobal && !isSuperAdmin(user.Role) {
		return "", ErrGlobalRuleForbidden
	}

	skill := &agentmodel.Skill{
		Name:      req.Name,
		Content:   req.Content,
		IsGlobal:  req.IsGlobal,
		IsEnabled: req.IsEnabled,
	}
	skill.UID = uuid.GenerateShortUUID()
	skill.OrgID = scopeOrgID(req.IsGlobal, user.Org.ID)
	skill.CreatedBy = user.User.ID
	skill.UpdatedBy = user.User.ID
	if err := s.db.Create(skill); err != nil {
		return "", err
	}
	return skill.UID, nil
}

// UpdateSkill replaces the mutable fields of an existing skill. name and content are
// required so an omitted field can never silently blank the skill.
func (s *skillService) UpdateSkill(ctx context.Context, uid string, req *agentapi.SkillRequest) error {
	if req.Name == "" || req.Content == "" {
		return fmt.Errorf("name and content are required")
	}
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}

	existing, err := s.GetSkillByUID(ctx, uid)
	if err != nil {
		return err
	}
	if existing.OrgID == agentmodel.GlobalRuleOrgID && !isSuperAdmin(user.Role) {
		return ErrGlobalRuleForbidden
	}

	updates := map[string]interface{}{
		"name":       req.Name,
		"content":    req.Content,
		"is_enabled": req.IsEnabled,
		"updated_by": user.User.ID,
	}
	if err := s.db.RawDB().Model(&agentmodel.Skill{}).
		Where("uid = ? and org_id = ?", uid, existing.OrgID).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// DeleteSkill removes a skill. Global-skill deletion requires super admin.
func (s *skillService) DeleteSkill(ctx context.Context, uid string) error {
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}
	existing, err := s.GetSkillByUID(ctx, uid)
	if err != nil {
		return err
	}
	if existing.OrgID == agentmodel.GlobalRuleOrgID && !isSuperAdmin(user.Role) {
		return ErrGlobalRuleForbidden
	}
	if err := s.db.Delete(&agentmodel.Skill{}, "uid = ? and org_id = ?", uid, existing.OrgID); err != nil {
		return err
	}
	return nil
}
