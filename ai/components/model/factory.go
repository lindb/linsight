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

package model

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"

	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/util"
)

// modelSettingsReader is the minimal interface that ModelFactory needs from the
// settings layer. Defined locally to avoid an import cycle with agent/service.
type modelSettingsReader interface {
	GetOrgModelConfig(ctx context.Context) (*agentmodel.OrgModelConfig, error)
	GetEnabledModels(ctx context.Context) ([]*agentmodel.Model, error)
	GetDefaultModel(ctx context.Context) (*agentmodel.Model, error)
}

// ErrNoDefaultModel mirrors service.ErrNoDefaultModel to avoid an import cycle.
// ModelFactory callers should check against this sentinel when mode is "fixed".
var ErrNoDefaultModel = errors.New("no default model configured for org")

// cacheEntry holds a compiled LLM client and the config hash it was built from.
// When the hash changes (org admin updated the model config) the entry is evicted.
type cacheEntry struct {
	hash  string
	model einomodel.ToolCallingChatModel
}

// ModelFactory builds and caches per-org LLM clients.
// It reads routing strategy and model configuration from the DB on each GetForOrg call,
// caching the resulting client by config hash so rebuilds happen only on config change.
type ModelFactory struct {
	settingsSrv modelSettingsReader
	// cache maps orgID (int64) → *cacheEntry
	cache sync.Map
}

// NewModelFactory creates a ModelFactory backed by settingsSrv for per-org model config.
func NewModelFactory(settingsSrv modelSettingsReader) *ModelFactory {
	return &ModelFactory{
		settingsSrv: settingsSrv,
	}
}

// GetForOrg resolves the active model for the given org using the org's routing strategy.
//
// "fixed" mode: returns the model with is_default=true; fast-fails with ErrNoDefaultModel
//               if none is configured.
// "auto" mode:  loads all enabled models and delegates to the configured ModelSelector;
//               fast-fails with ErrNoEnabledModels if the enabled list is empty.
//
// The resolved LLM client is cached by config hash and rebuilt only when the hash changes.
func (f *ModelFactory) GetForOrg(ctx context.Context, orgID int64) (einomodel.ToolCallingChatModel, error) {
	routingCfg, err := f.settingsSrv.GetOrgModelConfig(ctx)
	if err != nil {
		return nil, err
	}

	var selected *agentmodel.Model

	if routingCfg.SelectionMode == agentmodel.SelectionModeAuto {
		models, err := f.settingsSrv.GetEnabledModels(ctx)
		if err != nil {
			return nil, err
		}
		selector := newSelector(routingCfg.AutoStrategy)
		selected, err = selector.Select(ctx, models)
		if err != nil {
			return nil, err
		}
	} else {
		// Default to "fixed" mode.
		selected, err = f.settingsSrv.GetDefaultModel(ctx)
		if err != nil {
			return nil, ErrNoDefaultModel
		}
	}

	hash := selected.ConfigHash()

	// Fast path: return cached instance when hash matches.
	if v, ok := f.cache.Load(orgID); ok {
		entry := v.(*cacheEntry)
		if entry.hash == hash {
			return entry.model, nil
		}
	}

	// Slow path: build a new client and cache it.
	cfg := buildChatModelConfig(selected)
	m, err := openaimodel.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	f.cache.Store(orgID, &cacheEntry{hash: hash, model: m})
	return m, nil
}

// Invalidate evicts the cached model for orgID, forcing a rebuild on the next request.
// Call this after SaveOrgModelConfig or SetDefaultModel to ensure the new config is used.
func (f *ModelFactory) Invalidate(orgID int64) {
	f.cache.Delete(orgID)
}

// buildChatModelConfig converts an Model DB row to an openai ChatModelConfig.
// ExtraFields is parsed to inject model-specific thinking parameters:
//   - OpenAI o1/o3: {"reasoning_effort": "low|medium|high"}
//   - DeepSeek Reasoner: no extra params needed (reasoning_content is automatic)
func buildChatModelConfig(m *agentmodel.Model) *openaimodel.ChatModelConfig {
	cfg := &openaimodel.ChatModelConfig{
		Model:   m.Model,
		APIKey:  m.APIKey,
		BaseURL: m.BaseURL,
	}
	if m.MaxTokens > 0 {
		mt := m.MaxTokens
		cfg.MaxTokens = &mt
	}
	if m.Temperature != 0 {
		t := float32(m.Temperature)
		cfg.Temperature = &t
	}
	// Parse ExtraFields to inject model-specific thinking parameters.
	if len(m.ExtraFields) > 0 {
		var extra map[string]any
		if json.Unmarshal(m.ExtraFields, &extra) == nil {
			if re, ok := extra["reasoning_effort"].(string); ok && re != "" {
				cfg.ReasoningEffort = openaimodel.ReasoningEffortLevel(re)
			}
		}
	}
	return cfg
}

// DynamicModelProxy implements model.ToolCallingChatModel by delegating to the
// per-org model resolved from ModelFactory at call time.
// This allows the ReAct graph to be compiled once and reused across orgs.
type DynamicModelProxy struct {
	factory *ModelFactory
	// tools holds the bound tool schemas for the current WithTools call.
	tools []*schema.ToolInfo
	// orgID is set when the proxy is bound to a specific org via ForOrg.
	orgID int64
}

// NewDynamicModelProxy creates a proxy backed by the given factory.
func NewDynamicModelProxy(factory *ModelFactory) *DynamicModelProxy {
	return &DynamicModelProxy{factory: factory}
}

// ForOrg returns a new proxy bound to the given org ID.
func (p *DynamicModelProxy) ForOrg(orgID int64) *DynamicModelProxy {
	return &DynamicModelProxy{
		factory: p.factory,
		tools:   p.tools,
		orgID:   orgID,
	}
}

// WithTools returns a new proxy that passes tools to the underlying model.
// Implements model.ToolCallingChatModel.
func (p *DynamicModelProxy) WithTools(tools []*schema.ToolInfo) (einomodel.ToolCallingChatModel, error) {
	return &DynamicModelProxy{
		factory: p.factory,
		tools:   tools,
		orgID:   p.orgID,
	}, nil
}

// Generate delegates to the per-org model.
func (p *DynamicModelProxy) Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error) {
	m, err := p.resolveModel(ctx)
	if err != nil {
		return nil, err
	}
	return m.Generate(ctx, input, opts...)
}

// Stream delegates to the per-org model.
func (p *DynamicModelProxy) Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	m, err := p.resolveModel(ctx)
	if err != nil {
		return nil, err
	}
	return m.Stream(ctx, input, opts...)
}

// resolveModel fetches the real model from the factory, binding tools if any.
// orgID is resolved from the context (via the signed-in user) so that a single
// compiled graph can serve all orgs without rebuilding.
func (p *DynamicModelProxy) resolveModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	orgID := p.orgID
	if orgID == 0 {
		// Fall back to the request-scoped org from the authenticated user.
		if u := util.GetUser(ctx); u != nil && u.Org != nil {
			orgID = u.Org.ID
		}
	}
	base, err := p.factory.GetForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if len(p.tools) > 0 {
		return base.WithTools(p.tools)
	}
	return base, nil
}
