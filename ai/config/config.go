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

package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/lindb/common/pkg/ltoml"
)

// Config holds all configuration for the agent sub-module.
// Enabled must be the first field; when false, Bootstrap is a no-op.
// LLM model credentials (provider, API key, base URL, etc.) are managed entirely
// through the DB (agent_models table) and are no longer stored here.
type Config struct {
	// Enabled gates the entire agent feature. Default: false.
	Enabled bool `env:"ENABLED" toml:"enabled"`

	// MaxStep is the maximum number of reasoning steps before the agent
	// returns an error. Prevents infinite tool-call loops.
	MaxStep int `env:"MAX_STEP" toml:"max-step"`

	// RequestTimeout is the wall-clock deadline for an entire Chat() call,
	// including all LLM round-trips and tool invocations.
	RequestTimeout ltoml.Duration `env:"REQUEST_TIMEOUT" toml:"request-timeout"`

	// ToolTimeout is the per-tool-call deadline applied to each individual MCP
	// tool invocation. Must be less than RequestTimeout.
	ToolTimeout ltoml.Duration `env:"TOOL_TIMEOUT" toml:"tool-timeout"`

	// MaxConcurrentPerOrg caps concurrent in-flight Chat() calls per organisation.
	// Requests that exceed this limit are rejected with HTTP 429.
	MaxConcurrentPerOrg int `env:"MAX_CONCURRENT_PER_ORG" toml:"max-concurrent-per-org"`

	// MemorySize is the number of historical messages loaded into the LLM context
	// for each request (sliding window, newest first).
	MemorySize int `env:"MEMORY_SIZE" toml:"memory-size"`

	// MaxMessageLength is the maximum byte length of ChatRequest.Content.
	// Requests exceeding this are rejected with HTTP 400.
	MaxMessageLength int `env:"MAX_MESSAGE_LENGTH" toml:"max-message-length"`

	// MemoryStoreType specifies the storage backend for conversation state.
	// Options: "db" (default), "redis" (future)
	MemoryStoreType string `env:"MEMORY_STORE_TYPE" toml:"memory-store-type"`

	// MemoryStoreTTL specifies the time-to-live for conversation state cache.
	MemoryStoreTTL ltoml.Duration `env:"MEMORY_STORE_TTL" toml:"memory-store-ttl"`

	// EnableStateLocking enables conversation state locking to prevent concurrent access.
	EnableStateLocking bool `env:"ENABLE_STATE_LOCKING" toml:"enable-state-locking"`

	// PromptRigidityLevel controls the strictness of prompt constraints to prevent planning drift.
	// Options: "none", "basic", "strict" (default)
	PromptRigidityLevel string `env:"PROMPT_RIGIDITY_LEVEL" toml:"prompt-rigidity-level"`

	// EnableMultiTurnContext enables multi-turn conversation context preservation.
	EnableMultiTurnContext bool `env:"ENABLE_MULTI_TURN_CONTEXT" toml:"enable-multi-turn-context"`

	// MaxContextWindowSize specifies the maximum number of messages to include in context window.
	MaxContextWindowSize int `env:"MAX_CONTEXT_WINDOW_SIZE" toml:"max-context-window-size"`

	// RuleRefreshInterval is how often the DB-backed rule engine reloads its in-memory
	// cache from the agent_rules table. A non-positive value disables periodic refresh
	// (writes still trigger an immediate refresh). Default: 30s.
	RuleRefreshInterval ltoml.Duration `env:"RULE_REFRESH_INTERVAL" toml:"rule-refresh-interval"`

	// SkillRefreshInterval is how often the DB-backed skill backend reloads its in-memory
	// cache from the skills table. A non-positive value disables periodic refresh.
	// Default: 30s.
	SkillRefreshInterval ltoml.Duration `env:"SKILL_REFRESH_INTERVAL" toml:"skill-refresh-interval"`
}

// NewDefaultConfig returns a Config with conservative, safe defaults.
// All guards are enabled; the feature itself is disabled (Enabled: false).
func NewDefaultConfig() *Config {
	return &Config{
		Enabled:             false,
		MaxStep:             12,
		RequestTimeout:      ltoml.Duration(120 * time.Second),
		ToolTimeout:         ltoml.Duration(30 * time.Second),
		MaxConcurrentPerOrg: 5,
		MemorySize:          20,
		MaxMessageLength:    8192,

		// MemoryStore configuration defaults
		MemoryStoreType:        "db",
		MemoryStoreTTL:         ltoml.Duration(24 * time.Hour),
		EnableStateLocking:     true,
		PromptRigidityLevel:    "strict",
		EnableMultiTurnContext: true,
		MaxContextWindowSize:   50,
		RuleRefreshInterval:    ltoml.Duration(30 * time.Second),
		SkillRefreshInterval:   ltoml.Duration(30 * time.Second),
	}
}

// TOML returns the agent configuration as a commented TOML string fragment.
func (c *Config) TOML() string {
	return fmt.Sprintf(`
## Enable the AI agent feature. Default: %v
## Env: LINSIGHT_AI_ENABLED
enabled = %v

## Maximum reasoning steps before giving up. Default: %d
## Env: LINSIGHT_AI_MAX_STEP
max-step = %d

## Timeout for an entire chat request. Default: %s
## Env: LINSIGHT_AI_REQUEST_TIMEOUT
request-timeout = "%s"

## Timeout for a single MCP tool call. Default: %s
## Env: LINSIGHT_AI_TOOL_TIMEOUT
tool-timeout = "%s"

## Max concurrent chat requests per organisation. Excess requests return 429. Default: %d
## Env: LINSIGHT_AI_MAX_CONCURRENT_PER_ORG
max-concurrent-per-org = %d

## Number of historical messages to include in each request context. Default: %d
## Env: LINSIGHT_AI_MEMORY_SIZE
memory-size = %d

## Max byte length of the user message. Excess returns 400. Default: %d
## Env: LINSIGHT_AI_MAX_MESSAGE_LENGTH
max-message-length = %d

## Storage backend for conversation state. Options: "db" (default), "redis" (future)
## Env: LINSIGHT_AI_MEMORY_STORE_TYPE
memory-store-type = "%s"

## Time-to-live for conversation state cache. Default: %s
## Env: LINSIGHT_AI_MEMORY_STORE_TTL
memory-store-ttl = "%s"

## Enable conversation state locking to prevent concurrent access. Default: %v
## Env: LINSIGHT_AI_ENABLE_STATE_LOCKING
enable-state-locking = %v

## Strictness of prompt constraints to prevent planning drift. Options: "none", "basic", "strict" (default)
## Env: LINSIGHT_AI_PROMPT_RIGIDITY_LEVEL
prompt-rigidity-level = "%s"

## Enable multi-turn conversation context preservation. Default: %v
## Env: LINSIGHT_AI_ENABLE_MULTI_TURN_CONTEXT
enable-multi-turn-context = %v

## Maximum number of messages to include in context window. Default: %d
## Env: LINSIGHT_AI_MAX_CONTEXT_WINDOW_SIZE
max-context-window-size = %d

## How often the DB-backed rule engine reloads its in-memory cache. Default: %s
## Env: LINSIGHT_AI_RULE_REFRESH_INTERVAL
rule-refresh-interval = "%s"

	## How often the DB-backed skill backend reloads its in-memory cache. Default: %s
	## Env: LINSIGHT_AI_SKILL_REFRESH_INTERVAL
	skill-refresh-interval = "%s"`,
	c.Enabled, c.Enabled,
		c.MaxStep, c.MaxStep,
		c.RequestTimeout.Duration().String(), c.RequestTimeout.Duration().String(),
		c.ToolTimeout.Duration().String(), c.ToolTimeout.Duration().String(),
		c.MaxConcurrentPerOrg, c.MaxConcurrentPerOrg,
		c.MemorySize, c.MemorySize,
		c.MaxMessageLength, c.MaxMessageLength,
		escapeTOMLString(c.MemoryStoreType),
		c.MemoryStoreTTL.Duration().String(), c.MemoryStoreTTL.Duration().String(),
		c.EnableStateLocking, c.EnableStateLocking,
		escapeTOMLString(c.PromptRigidityLevel),
		c.EnableMultiTurnContext, c.EnableMultiTurnContext,
		c.MaxContextWindowSize, c.MaxContextWindowSize,
		c.RuleRefreshInterval.Duration().String(), c.RuleRefreshInterval.Duration().String(),
		c.SkillRefreshInterval.Duration().String(), c.SkillRefreshInterval.Duration().String(),
	)
}

// escapeTOMLString escapes special characters in s for safe embedding in a TOML basic string.
// TOML basic strings are delimited by double quotes and do not allow literal newlines or unescaped quotes.
func escapeTOMLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
