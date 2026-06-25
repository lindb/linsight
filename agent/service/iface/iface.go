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

package iface

import (
	"context"

	agentapi "github.com/lindb/linsight/agent/api"
	agentmodel "github.com/lindb/linsight/agent/model"
)

// AgentSettingsService abstracts all DB-backed agent settings operations.
// Implemented by service.AgentSettingsService; this interface lives in a
// separate package so model/factory and tools/mcp can import it without
// creating a circular dependency with the service package.
//
// All methods extract org/user IDs from ctx internally — callers must never
// pass orgID/userID as parameters.
type AgentSettingsService interface {
	// --- Model CRUD ---

	// ListModels returns all named model configurations for the org in ctx.
	ListModels(ctx context.Context) ([]*agentmodel.AgentModel, error)
	// GetModelByUID returns a single model configuration by UID within the org in ctx.
	GetModelByUID(ctx context.Context, uid string) (*agentmodel.AgentModel, error)
	// GetDefaultModel returns the model with is_default=true for the org in ctx.
	// Returns ErrNoDefaultModel if no default has been configured.
	GetDefaultModel(ctx context.Context) (*agentmodel.AgentModel, error)
	// GetEnabledModels returns all models with is_enabled=true for the org in ctx,
	// ordered by priority asc, cost_weight asc for deterministic selector behaviour.
	GetEnabledModels(ctx context.Context) ([]*agentmodel.AgentModel, error)
	// CreateModel inserts a new named model configuration.
	// Returns the generated UID.
	CreateModel(ctx context.Context, req *agentapi.CreateModelRequest) (string, error)
	// UpdateModel applies a partial update to an existing model configuration.
	UpdateModel(ctx context.Context, uid string, req *agentapi.UpdateModelRequest) error
	// DeleteModel removes a model configuration by UID within the org in ctx.
	DeleteModel(ctx context.Context, uid string) error
	// SetDefaultModel atomically clears is_default on all org models, then sets
	// is_default=true on the model identified by uid.
	SetDefaultModel(ctx context.Context, uid string) error

	// --- Routing config ---

	// GetOrgModelConfig returns the per-org routing strategy configuration.
	// Returns a default (SelectionModeFixed, no strategy) if no row exists.
	GetOrgModelConfig(ctx context.Context) (*agentmodel.AgentOrgModelConfig, error)
	// SaveOrgModelConfig creates or updates the per-org routing strategy.
	SaveOrgModelConfig(ctx context.Context, req *agentapi.SaveOrgModelConfigRequest) error

	// --- MCP server settings ---

	// ListMCPServers returns all MCP server entries for the org in ctx.
	ListMCPServers(ctx context.Context) ([]*agentmodel.AgentMCPServer, error)
	// GetMCPServer returns a single MCP server by UID within the org in ctx.
	GetMCPServer(ctx context.Context, uid string) (*agentmodel.AgentMCPServer, error)
	// CreateMCPServer inserts a new MCP server configuration.
	// Returns the generated UID.
	CreateMCPServer(ctx context.Context, req *agentapi.CreateMCPServerRequest) (string, error)
	// DeleteMCPServer removes an MCP server configuration by UID within the org in ctx.
	DeleteMCPServer(ctx context.Context, uid string) error

	// --- Conversations ---

	// GetOrCreateConversation returns the conversation with the given UID, or creates
	// a new one if uid is empty. Returns the conversation and whether it was just created.
	GetOrCreateConversation(ctx context.Context, uid string) (*agentmodel.AgentConversation, bool, error)
	// ListConversations returns paginated conversations for the user in ctx, newest first.
	ListConversations(ctx context.Context, offset, limit int) ([]*agentmodel.AgentConversation, int64, error)
	// DeleteConversation removes a conversation and all its messages.
	DeleteConversation(ctx context.Context, uid string) error

	// --- Messages ---

	// GetRecentMessages returns up to limit messages for the conversation, ordered
	// by ID descending (newest first), then reversed so the oldest is at index 0.
	GetRecentMessages(ctx context.Context, conversationID int64, limit int) ([]*agentmodel.AgentMessage, error)
	// SaveMessage persists a single message to the conversation.
	SaveMessage(ctx context.Context, msg *agentmodel.AgentMessage) error
}
