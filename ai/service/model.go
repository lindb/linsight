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

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ErrNoDefaultModel is returned when GetDefaultModel finds no is_default=true row.
var ErrNoDefaultModel = errors.New("no default model configured for org")

// ModelService manages AI model configurations and per-org routing strategy.
type ModelService interface {
	// ListModels returns all named model configurations for the org in ctx.
	ListModels(ctx context.Context) ([]*agentmodel.Model, error)
	// GetModelByUID returns a single model configuration by UID within the org in ctx.
	GetModelByUID(ctx context.Context, uid string) (*agentmodel.Model, error)
	// GetDefaultModel returns the model with is_default=true and is_enabled=true for the org in ctx.
	// Returns ErrNoDefaultModel if no default has been configured or the default model is disabled.
	GetDefaultModel(ctx context.Context) (*agentmodel.Model, error)
	// GetEnabledModels returns all models with is_enabled=true for the org in ctx,
	// ordered by priority asc, cost_weight asc for deterministic selector behaviour.
	GetEnabledModels(ctx context.Context) ([]*agentmodel.Model, error)
	// CreateModel inserts a new named model configuration.
	// Returns the generated UID. Required fields are validated by the service layer.
	CreateModel(ctx context.Context, req *agentapi.ModelRequest) (string, error)
	// UpdateModel applies a partial update to an existing model configuration.
	UpdateModel(ctx context.Context, uid string, req *agentapi.ModelRequest) error
	// DeleteModel removes a model configuration by UID within the org in ctx.
	DeleteModel(ctx context.Context, uid string) error
	// SetDefaultModel atomically clears is_default on all org models, then sets
	// is_default=true on the model identified by uid.
	SetDefaultModel(ctx context.Context, uid string) error
	// GetOrgModelConfig returns the per-org routing strategy configuration.
	GetOrgModelConfig(ctx context.Context) (*agentmodel.OrgModelConfig, error)
	// SaveOrgModelConfig creates or updates the per-org routing strategy.
	SaveOrgModelConfig(ctx context.Context, req *agentapi.SaveOrgModelConfigRequest) error
}

type modelService struct {
	db dbpkg.DB
}

// NewModelService creates an ModelService backed by db.
func NewModelService(db dbpkg.DB) ModelService {
	return &modelService{db: db}
}

// ── Model CRUD ────────────────────────────────────────────────────────────────

// ListModels returns all named model configurations for the org extracted from ctx.
func (s *modelService) ListModels(ctx context.Context) ([]*agentmodel.Model, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var models []*agentmodel.Model
	if err := s.db.RawDB().Where("org_id = ?", orgID).Order("priority asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	return models, nil
}

// GetModelByUID returns a single model configuration by UID within the org extracted from ctx.
func (s *modelService) GetModelByUID(ctx context.Context, uid string) (*agentmodel.Model, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var m agentmodel.Model
	if err := s.db.Get(&m, "org_id = ? and uid = ?", orgID, uid); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetDefaultModel returns the model with is_default=true and is_enabled=true for the org extracted from ctx.
// Returns ErrNoDefaultModel if no default has been configured or the default model is disabled.
func (s *modelService) GetDefaultModel(ctx context.Context) (*agentmodel.Model, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var m agentmodel.Model
	if err := s.db.Get(&m, "org_id = ? and is_default = ? and is_enabled = ?", orgID, true, true); err != nil {
		return nil, ErrNoDefaultModel
	}
	return &m, nil
}

// GetEnabledModels returns all models with is_enabled=true for the org extracted from ctx,
// ordered by priority asc, cost_weight asc for deterministic selector behaviour.
func (s *modelService) GetEnabledModels(ctx context.Context) ([]*agentmodel.Model, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var models []*agentmodel.Model
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
// Name, Provider, Model, and APIKey are required; validation is done here
// rather than in gin binding so service callers outside HTTP also get checks.
func (s *modelService) CreateModel(ctx context.Context, req *agentapi.ModelRequest) (string, error) {
	if req.Name == "" || req.Provider == "" || req.Model == "" || req.APIKey == "" {
		return "", fmt.Errorf("name, provider, model, and apiKey are required")
	}
	user := util.GetUser(ctx)
	model := &agentmodel.Model{
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
	// If the caller requested this model to be the default, promote it atomically.
	// SetDefaultModel clears is_default on all other org models in a transaction.
	if req.IsDefault {
		if err := s.SetDefaultModel(ctx, model.UID); err != nil {
			return "", fmt.Errorf("set default model: %w", err)
		}
	}
	return model.UID, nil
}

// UpdateModel applies a partial update to an existing model configuration from req.
// APIKey is only overwritten when req.APIKey is non-empty; an empty value leaves
// the stored key untouched so callers do not have to re-submit the key on every edit.
func (s *modelService) UpdateModel(ctx context.Context, uid string, req *agentapi.ModelRequest) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	updates := map[string]interface{}{
		"name":        req.Name,
		"provider":    req.Provider,
		"model":       req.Model,
		"base_url":    req.BaseURL,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"is_enabled":  req.IsEnabled,
		"is_default":  req.IsDefault,
		"priority":    req.Priority,
		"cost_weight": req.CostWeight,
		"tags":        req.Tags,
		"updated_by":  user.User.ID,
	}
	if req.APIKey != "" {
		updates["api_key"] = req.APIKey
	}
	return s.db.RawDB().Model(&agentmodel.Model{}).
		Where("org_id = ? and uid = ?", orgID, uid).
		Updates(updates).Error
}

// DeleteModel removes a model configuration by UID within the org extracted from ctx.
func (s *modelService) DeleteModel(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Delete(&agentmodel.Model{}, "org_id = ? and uid = ?", orgID, uid)
}

// SetDefaultModel atomically clears is_default on all org models, then sets
// is_default=true on the model identified by uid.
func (s *modelService) SetDefaultModel(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Transaction(func(tx dbpkg.DB) error {
		// Clear all defaults for the org.
		if err := tx.RawDB().Model(&agentmodel.Model{}).
			Where("org_id = ?", orgID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		// Set the target model as default.
		return tx.RawDB().Model(&agentmodel.Model{}).
			Where("org_id = ? and uid = ?", orgID, uid).
			Update("is_default", true).Error
	})
}

// ── Routing config ────────────────────────────────────────────────────────────

// GetOrgModelConfig returns the per-org routing strategy configuration for the org in ctx.
// Returns a default (SelectionModeFixed, no strategy) if no row exists.
func (s *modelService) GetOrgModelConfig(ctx context.Context) (*agentmodel.OrgModelConfig, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	var cfg agentmodel.OrgModelConfig
	if err := s.db.Get(&cfg, "org_id = ?", orgID); err != nil {
		// No row yet — return a safe default (fixed mode, no auto strategy).
		return &agentmodel.OrgModelConfig{
			OrgID:         orgID,
			SelectionMode: agentmodel.SelectionModeFixed,
		}, nil
	}
	return &cfg, nil
}

// SaveOrgModelConfig creates or updates the per-org routing strategy from req.
func (s *modelService) SaveOrgModelConfig(ctx context.Context, req *agentapi.SaveOrgModelConfigRequest) error {
	user := util.GetUser(ctx)
	cfg := &agentmodel.OrgModelConfig{
		OrgID:         user.Org.ID,
		SelectionMode: agentmodel.SelectionMode(req.SelectionMode),
		AutoStrategy:  agentmodel.AutoStrategy(req.AutoStrategy),
	}

	var existing agentmodel.OrgModelConfig
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
