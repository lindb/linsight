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

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/uuid"
)

// SpecialistService manages Specialist (sub-agent) CRUD with two-tier
// (global + per-org) scoping. It mirrors SkillService: the service touches the DB
// only — cache freshness is maintained by multiagent.DBBackend's background ticker.
type SpecialistService interface {
	// ListSpecialists returns global specialists plus those owned by the org in ctx.
	ListSpecialists(ctx context.Context) ([]*agentmodel.Specialist, error)
	// GetSpecialistByUID returns a single specialist visible to the org in ctx (own or global).
	GetSpecialistByUID(ctx context.Context, uid string) (*agentmodel.Specialist, error)
	// CreateSpecialist inserts a new specialist and returns its generated UID.
	// Creating a global specialist requires super-admin (RoleLin).
	CreateSpecialist(ctx context.Context, req *agentapi.AgentRequest) (string, error)
	// UpdateSpecialist applies a full replace update within the caller's scope.
	UpdateSpecialist(ctx context.Context, uid string, req *agentapi.AgentRequest) error
	// DeleteSpecialist removes a specialist within the caller's scope.
	DeleteSpecialist(ctx context.Context, uid string) error
}

type specialistService struct {
	db dbpkg.DB
}

// NewSpecialistService creates a SpecialistService backed by db. The service does not
// interact with the specialist cache — freshness is maintained by DBBackend's ticker.
func NewSpecialistService(db dbpkg.DB) SpecialistService {
	return &specialistService{db: db}
}

// ListSpecialists returns global specialists (org_id=0) plus those owned by the caller's org.
func (s *specialistService) ListSpecialists(ctx context.Context) ([]*agentmodel.Specialist, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var specialists []*agentmodel.Specialist
	if err := s.db.RawDB().
		Where("org_id = ? or org_id = ?", agentmodel.GlobalRuleOrgID, orgID).
		Order("name asc").
		Find(&specialists).Error; err != nil {
		return nil, err
	}
	return specialists, nil
}

// GetSpecialistByUID returns a specialist the caller may see: their org's or a global one.
func (s *specialistService) GetSpecialistByUID(ctx context.Context, uid string) (*agentmodel.Specialist, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}
	orgID := user.Org.ID
	var specialist agentmodel.Specialist
	if err := s.db.Get(&specialist, "uid = ? and (org_id = ? or org_id = ?)",
		uid, agentmodel.GlobalRuleOrgID, orgID); err != nil {
		return nil, err
	}
	return &specialist, nil
}

// CreateSpecialist builds a Specialist from req, enforcing the super-admin gate for
// global specialists and validating the name is a tool-name-safe slug.
func (s *specialistService) CreateSpecialist(ctx context.Context, req *agentapi.AgentRequest) (string, error) {
	if req.Name == "" || req.Content == "" {
		return "", fmt.Errorf("name and content are required")
	}
	if err := agentmodel.ValidateName(req.Name); err != nil {
		return "", err
	}
	user, err := requireUser(ctx)
	if err != nil {
		return "", err
	}
	if req.IsGlobal && !isSuperAdmin(user.Role) {
		return "", ErrGlobalRuleForbidden
	}

	specialist := &agentmodel.Specialist{
		Name:      req.Name,
		Content:   req.Content,
		IsGlobal:  req.IsGlobal,
		IsEnabled: req.IsEnabled,
	}
	specialist.UID = uuid.GenerateShortUUID()
	specialist.OrgID = scopeOrgID(req.IsGlobal, user.Org.ID)
	specialist.CreatedBy = user.User.ID
	specialist.UpdatedBy = user.User.ID
	if err := s.db.Create(specialist); err != nil {
		return "", err
	}
	return specialist.UID, nil
}

// UpdateSpecialist replaces the mutable fields of an existing specialist. name and
// content are required so an omitted field can never silently blank them. The
// scope (org vs global) is immutable, so IsGlobal is ignored on update.
func (s *specialistService) UpdateSpecialist(ctx context.Context, uid string, req *agentapi.AgentRequest) error {
	if req.Name == "" || req.Content == "" {
		return fmt.Errorf("name and content are required")
	}
	if err := agentmodel.ValidateName(req.Name); err != nil {
		return err
	}
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}

	existing, err := s.GetSpecialistByUID(ctx, uid)
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
	if err := s.db.RawDB().Model(&agentmodel.Specialist{}).
		Where("uid = ? and org_id = ?", uid, existing.OrgID).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// DeleteSpecialist removes a specialist. Global-specialist deletion requires super admin;
// per-org deletion is constrained to the caller's org.
func (s *specialistService) DeleteSpecialist(ctx context.Context, uid string) error {
	user, err := requireUser(ctx)
	if err != nil {
		return err
	}
	existing, err := s.GetSpecialistByUID(ctx, uid)
	if err != nil {
		return err
	}
	if existing.OrgID == agentmodel.GlobalRuleOrgID && !isSuperAdmin(user.Role) {
		return ErrGlobalRuleForbidden
	}
	if err := s.db.Delete(&agentmodel.Specialist{}, "uid = ? and org_id = ?", uid, existing.OrgID); err != nil {
		return err
	}
	return nil
}
