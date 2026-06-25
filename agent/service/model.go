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

	agentapi "github.com/lindb/linsight/agent/api"
	agentmodel "github.com/lindb/linsight/agent/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ── Model CRUD ────────────────────────────────────────────────────────────────

// ListModels returns all named model configurations for the org extracted from ctx.
func (s *agentSettingsService) ListModels(ctx context.Context) ([]*agentmodel.AgentModel, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var models []*agentmodel.AgentModel
	if err := s.db.RawDB().Where("org_id = ?", orgID).Order("priority asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// GetModelByUID returns a single model configuration by UID within the org extracted from ctx.
func (s *agentSettingsService) GetModelByUID(ctx context.Context, uid string) (*agentmodel.AgentModel, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var m agentmodel.AgentModel
	if err := s.db.Get(&m, "org_id = ? and uid = ?", orgID, uid); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetDefaultModel returns the model with is_default=true for the org extracted from ctx.
// Returns ErrNoDefaultModel if no default has been configured.
func (s *agentSettingsService) GetDefaultModel(ctx context.Context) (*agentmodel.AgentModel, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var m agentmodel.AgentModel
	if err := s.db.Get(&m, "org_id = ? and is_default = ?", orgID, true); err != nil {
		return nil, ErrNoDefaultModel
	}
	return &m, nil
}

// GetEnabledModels returns all models with is_enabled=true for the org extracted from ctx,
// ordered by priority asc, cost_weight asc for deterministic selector behaviour.
func (s *agentSettingsService) GetEnabledModels(ctx context.Context) ([]*agentmodel.AgentModel, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var models []*agentmodel.AgentModel
	if err := s.db.RawDB().
		Where("org_id = ? and is_enabled = ?", orgID, true).
		Order("priority asc, cost_weight asc").
		Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// CreateModel inserts a new named model configuration built from req.
// Returns the generated UID.
func (s *agentSettingsService) CreateModel(ctx context.Context, req *agentapi.CreateModelRequest) (string, error) {
	user := util.GetUser(ctx)
	model := &agentmodel.AgentModel{
		Name:        req.Name,
		Provider:    req.Provider,
		Model:       req.Model,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		IsEnabled:   req.IsEnabled,
		Priority:    req.Priority,
		CostWeight:  req.CostWeight,
		Tags:        req.Tags,
	}
	model.UID = uuid.GenerateShortUUID()
	model.OrgID = user.Org.ID
	model.CreatedBy = user.User.ID
	model.UpdatedBy = user.User.ID
	if err := s.db.Create(model); err != nil {
		return "", err
	}
	return model.UID, nil
}

// UpdateModel applies a partial update to an existing model configuration from req.
func (s *agentSettingsService) UpdateModel(ctx context.Context, uid string, req *agentapi.UpdateModelRequest) error {
	user := util.GetUser(ctx)
	model := &agentmodel.AgentModel{
		UID:         uid,
		OrgID:       user.Org.ID,
		Name:        req.Name,
		Provider:    req.Provider,
		Model:       req.Model,
		APIKey:      req.APIKey,
		BaseURL:     req.BaseURL,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		IsEnabled:   req.IsEnabled,
		IsDefault:   req.IsDefault,
		Priority:    req.Priority,
		CostWeight:  req.CostWeight,
		Tags:        req.Tags,
	}
	model.UpdatedBy = user.User.ID
	return s.db.Updates(&agentmodel.AgentModel{}, model, "org_id = ? and uid = ?", model.OrgID, model.UID)
}

// DeleteModel removes a model configuration by UID within the org extracted from ctx.
func (s *agentSettingsService) DeleteModel(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Delete(&agentmodel.AgentModel{}, "org_id = ? and uid = ?", orgID, uid)
}

// SetDefaultModel atomically clears is_default on all org models, then sets
// is_default=true on the model identified by uid.
func (s *agentSettingsService) SetDefaultModel(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Transaction(func(tx dbpkg.DB) error {
		// Clear all defaults for the org.
		if err := tx.RawDB().Model(&agentmodel.AgentModel{}).
			Where("org_id = ?", orgID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		// Set the target model as default.
		return tx.RawDB().Model(&agentmodel.AgentModel{}).
			Where("org_id = ? and uid = ?", orgID, uid).
			Update("is_default", true).Error
	})
}

// ── Routing config ────────────────────────────────────────────────────────────

// GetOrgModelConfig returns the per-org routing strategy configuration for the org in ctx.
// Returns a default (SelectionModeFixed, no strategy) if no row exists.
func (s *agentSettingsService) GetOrgModelConfig(ctx context.Context) (*agentmodel.AgentOrgModelConfig, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var cfg agentmodel.AgentOrgModelConfig
	if err := s.db.Get(&cfg, "org_id = ?", orgID); err != nil {
		// No row yet — return a safe default (fixed mode, no auto strategy).
		return &agentmodel.AgentOrgModelConfig{
			OrgID:         orgID,
			SelectionMode: agentmodel.SelectionModeFixed,
		}, nil
	}
	return &cfg, nil
}

// SaveOrgModelConfig creates or updates the per-org routing strategy from req.
func (s *agentSettingsService) SaveOrgModelConfig(ctx context.Context, req *agentapi.SaveOrgModelConfigRequest) error {
	user := util.GetUser(ctx)
	cfg := &agentmodel.AgentOrgModelConfig{
		OrgID:         user.Org.ID,
		SelectionMode: agentmodel.SelectionMode(req.SelectionMode),
		AutoStrategy:  agentmodel.AutoStrategy(req.AutoStrategy),
	}

	var existing agentmodel.AgentOrgModelConfig
	exists, err := s.db.Exist(&existing, "org_id = ?", cfg.OrgID)
	if err != nil {
		return err
	}
	if exists {
		cfg.CreatedBy = existing.CreatedBy
		cfg.UpdatedBy = user.User.ID
		return s.db.Updates(&existing, cfg, "org_id = ?", cfg.OrgID)
	}
	cfg.CreatedBy = user.User.ID
	cfg.UpdatedBy = user.User.ID
	return s.db.Create(cfg)
}
