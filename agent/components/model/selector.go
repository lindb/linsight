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
	"errors"
	"strings"
	"sync/atomic"

	agentmodel "github.com/lindb/linsight/agent/model"
)

// contextKey is the key type used by selector context helpers in this package.
type contextKey int

const capabilitiesKey contextKey = iota

// WithRequestCapabilities stores the required capability labels in ctx.
// CapabilitySelector reads these to find a model whose Tags cover all requested capabilities.
// Labels are matched case-insensitively against comma-separated model Tags.
func WithRequestCapabilities(ctx context.Context, caps []string) context.Context {
	return context.WithValue(ctx, capabilitiesKey, caps)
}

// getRequestCapabilities retrieves the capability labels stored by WithRequestCapabilities.
func getRequestCapabilities(ctx context.Context) []string {
	v, _ := ctx.Value(capabilitiesKey).([]string)
	return v
}

// ErrNoEnabledModels is returned when the auto selector finds no enabled models.
var ErrNoEnabledModels = errors.New("no enabled models configured for org")

// ModelSelector picks one AgentModel from a candidate slice.
type ModelSelector interface {
	Select(ctx context.Context, models []*agentmodel.AgentModel) (*agentmodel.AgentModel, error)
}

// ── FailoverSelector ──────────────────────────────────────────────────────────

// FailoverSelector returns the first model in the slice (assumed pre-sorted by priority asc).
// Actual retry-on-failure is handled at the LLM call layer; this selector just
// picks the primary model.
type FailoverSelector struct{}

func (FailoverSelector) Select(_ context.Context, models []*agentmodel.AgentModel) (*agentmodel.AgentModel, error) {
	if len(models) == 0 {
		return nil, ErrNoEnabledModels
	}
	return models[0], nil
}

// ── CostSelector ─────────────────────────────────────────────────────────────

// CostSelector returns the model with the lowest CostWeight (assumed pre-sorted by cost_weight asc).
type CostSelector struct{}

func (CostSelector) Select(_ context.Context, models []*agentmodel.AgentModel) (*agentmodel.AgentModel, error) {
	if len(models) == 0 {
		return nil, ErrNoEnabledModels
	}
	// Models already ordered by cost_weight asc from GetEnabledModels.
	return models[0], nil
}

// ── RoundRobinSelector ────────────────────────────────────────────────────────

// RoundRobinSelector distributes requests across models in round-robin order.
// The counter is per-instance and resets on server restart (acceptable — no persistence needed).
type RoundRobinSelector struct {
	counter atomic.Uint64
}

func (r *RoundRobinSelector) Select(_ context.Context, models []*agentmodel.AgentModel) (*agentmodel.AgentModel, error) {
	if len(models) == 0 {
		return nil, ErrNoEnabledModels
	}
	idx := r.counter.Add(1) - 1
	return models[idx%uint64(len(models))], nil
}

// ── CapabilitySelector ────────────────────────────────────────────────────────

// CapabilitySelector picks the first model whose Tags cover all capabilities
// requested via WithRequestCapabilities. Falls back to FailoverSelector when no
// capabilities are requested or when no model matches.
type CapabilitySelector struct {
	fallback FailoverSelector
}

func (c CapabilitySelector) Select(ctx context.Context, models []*agentmodel.AgentModel) (*agentmodel.AgentModel, error) {
	if len(models) == 0 {
		return nil, ErrNoEnabledModels
	}

	required := getRequestCapabilities(ctx)
	if len(required) == 0 {
		// No specific capabilities requested — delegate to failover.
		return c.fallback.Select(ctx, models)
	}

	for _, m := range models {
		if modelHasCapabilities(m.Tags, required) {
			return m, nil
		}
	}
	// No model matched required capabilities — fall back to priority order.
	return c.fallback.Select(ctx, models)
}

// modelHasCapabilities returns true when all required labels appear in the model's
// comma-separated Tags field (case-insensitive).
func modelHasCapabilities(tags string, required []string) bool {
	if tags == "" {
		return false
	}
	modelTags := make(map[string]struct{})
	for _, t := range strings.Split(tags, ",") {
		modelTags[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	for _, r := range required {
		if _, ok := modelTags[strings.ToLower(strings.TrimSpace(r))]; !ok {
			return false
		}
	}
	return true
}

// newSelector returns the ModelSelector implementation for the given AutoStrategy.
// Unknown strategies fall back to FailoverSelector.
func newSelector(strategy agentmodel.AutoStrategy) ModelSelector {
	switch strategy {
	case agentmodel.AutoStrategyCost:
		return CostSelector{}
	case agentmodel.AutoStrategyRoundRobin:
		return &RoundRobinSelector{}
	case agentmodel.AutoStrategyCapability:
		return CapabilitySelector{}
	default:
		// AutoStrategyFailover and anything unrecognised → priority-order failover.
		return FailoverSelector{}
	}
}
