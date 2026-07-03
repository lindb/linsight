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
	"crypto/sha256"
	"encoding/json"
	"fmt"

	basemodel "github.com/lindb/linsight/model"
)

// SelectionMode controls how ModelFactory picks a model for a given org.
type SelectionMode string

const (
	// SelectionModeFixed means the model with is_default=true is always used.
	SelectionModeFixed SelectionMode = "fixed"
	// SelectionModeAuto means the system picks based on an AutoStrategy.
	SelectionModeAuto SelectionMode = "auto"
)

// AutoStrategy is the algorithm used when SelectionMode is "auto".
type AutoStrategy string

const (
	// AutoStrategyFailover tries models in ascending priority order.
	AutoStrategyFailover AutoStrategy = "failover"
	// AutoStrategyCost picks the cheapest enabled model (lowest cost_weight).
	AutoStrategyCost AutoStrategy = "cost"
	// AutoStrategyRoundRobin distributes load across enabled models.
	AutoStrategyRoundRobin AutoStrategy = "round_robin"
	// AutoStrategyCapability matches model tags to the request's required capabilities.
	AutoStrategyCapability AutoStrategy = "capability"
)

// Model is a named LLM configuration entry. An org may have many rows.
// In "fixed" mode the row with is_default=true is used.
// In "auto" mode all rows with is_enabled=true are candidates for the selector.
type Model struct {
	basemodel.BaseModel

	UID   string `json:"uid" gorm:"column:uid;uniqueIndex"`
	// OrgID + Name form a composite unique index: duplicate names within the same org are rejected at DB level.
	OrgID int64  `json:"orgId" gorm:"column:org_id;uniqueIndex:idx_org_model_name"`
	// Name is a human-readable label for this model configuration (e.g. "GPT-4o Primary").
	Name     string  `json:"name" gorm:"column:name;uniqueIndex:idx_org_model_name"`
	Provider string  `json:"provider" gorm:"column:provider"`
	Model    string  `json:"model" gorm:"column:model"`
	APIKey   string  `json:"-" gorm:"column:api_key"`
	BaseURL  string  `json:"baseUrl" gorm:"column:base_url"`
	MaxTokens   int     `json:"maxTokens" gorm:"column:max_tokens"`
	Temperature float64 `json:"temperature" gorm:"column:temperature"`

	// IsDefault marks the model chosen when SelectionMode is "fixed".
	// At most one row per org should have is_default=true; enforced by SetDefaultModel.
	IsDefault bool `json:"isDefault" gorm:"column:is_default"`

	// IsEnabled controls whether this model participates in "auto" selection.
	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled"`

	// Priority is used by FailoverSelector: lower values are tried first.
	Priority int `json:"priority" gorm:"column:priority"`

	// CostWeight is used by CostSelector: lower values represent cheaper models.
	CostWeight int `json:"costWeight" gorm:"column:cost_weight"`

	// Tags is a comma-separated list of capability labels (e.g. "code,vision").
	// Used by CapabilitySelector to match request capabilities.
	Tags string `json:"tags" gorm:"column:tags"`

	// ExtraFields carries model-specific thinking parameters as JSON.
	// For OpenAI o1/o3: {"reasoning_effort": "low|medium|high"}
	// For DeepSeek Reasoner: empty (reasoning_content is returned automatically).
	// ModelFactory parses this at client build time and injects into ChatModelConfig.
	ExtraFields json.RawMessage `json:"extraFields" gorm:"column:extra_fields;type:json"`
}

// TableName returns the GORM table name.
func (Model) TableName() string { return "models" }

// ConfigHash returns a short fingerprint of the fields that determine the LLM client.
// ModelFactory uses this to decide whether a cached client can be reused.
// ExtraFields is normalised before hashing to avoid false mismatches from whitespace
// differences in the stored JSON (e.g. {"k":"v"} vs { "k" : "v" }).
func (m *Model) ConfigHash() string {
	extra := m.ExtraFields
	if len(extra) > 0 {
		var norm any
		if json.Unmarshal(extra, &norm) == nil {
			if b, err := json.Marshal(norm); err == nil {
				extra = b
			}
		}
	}
	raw := fmt.Sprintf("%s|%s|%s|%s|%d|%g|%s",
		m.Provider, m.Model, m.APIKey, m.BaseURL, m.MaxTokens, m.Temperature,
		string(extra))
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:8])
}

// OrgModelConfig stores the per-org routing strategy. One row per org.
// If no row exists, ModelFactory defaults to SelectionModeFixed.
type OrgModelConfig struct {
	basemodel.BaseModel

	OrgID         int64         `json:"orgId" gorm:"column:org_id;uniqueIndex"`
	SelectionMode SelectionMode `json:"selectionMode" gorm:"column:selection_mode"`
	// AutoStrategy is only meaningful when SelectionMode is "auto".
	AutoStrategy AutoStrategy `json:"autoStrategy" gorm:"column:auto_strategy"`
}

// TableName returns the GORM table name.
func (OrgModelConfig) TableName() string { return "org_model_configs" }

// MCPServerType indicates how the MCP server process is started.
type MCPServerType string

const (
	// MCPServerTypeSSE connects to an already-running MCP server via Server-Sent Events.
	MCPServerTypeSSE MCPServerType = "sse"
	// MCPServerTypeStdio spawns a subprocess and communicates over its stdin/stdout.
	MCPServerTypeStdio MCPServerType = "stdio"
)

// MCP represents a per-org MCP server configuration entry.
type MCP struct {
	basemodel.BaseModel
	UID     string        `json:"uid" gorm:"column:uid;uniqueIndex"`
	OrgID   int64         `json:"orgId" gorm:"column:org_id;index"`
	Name    string        `json:"name" gorm:"column:name"`
	Type    MCPServerType `json:"type" gorm:"column:type"`
	// URL is used for SSE servers (e.g. "http://localhost:3000/mcp/sse").
	URL     string        `json:"url" gorm:"column:url"`
	// Command is used for stdio servers (the executable path).
	Command string        `json:"command" gorm:"column:command"`
	// Args are the command-line arguments for stdio servers.
	Args    string        `json:"args" gorm:"column:args;type:text"`
	Enabled bool          `json:"enabled" gorm:"column:enabled;default:true"`
}

// TableName specifies the GORM table name.
func (MCP) TableName() string { return "mcps" }

// ConfigHash returns a fingerprint of the connection-relevant fields.
// A change in this hash invalidates the cached tool list in MCPToolsFactory.
func (s *MCP) ConfigHash() string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%v", s.Type, s.URL, s.Command, s.Args, s.Enabled)
	sum := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", sum[:8])
}
