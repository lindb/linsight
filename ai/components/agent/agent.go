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

// Package agent provides the ReAct streaming agent runtime.
// It is intentionally separate from ai/service/ (which handles management CRUD)
// to keep streaming orchestration concerns isolated.
package agent

import (
	"context"
	"fmt"

	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	rulescomp "github.com/lindb/linsight/ai/components/rules"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentservice "github.com/lindb/linsight/ai/service"
)

// Agent orchestrates LLM streaming chat with per-org model routing and AG-UI SSE delivery.
type Agent interface {
	// Chat runs one conversation turn, streaming AG-UI events to writer.
	// convUID identifies an existing conversation; an empty string creates a new one.
	// Returns the conversation UID, the run ID (always set, even on early failure), and any error.
	Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (string, string, error)
}

// New creates an Agent dispatching to the correct mode based on cfg.AgentMode.
// The ReAct graph is compiled eagerly at construction time using context.Background()
// so that any graph-build failure surfaces at startup rather than on the first request.
func New(
	cfg *agentconfig.Config,
	convSrv agentservice.ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
	ruleEngine rulescomp.RuleEngine,
) (Agent, error) {
	switch cfg.AgentMode {
	case agentconfig.AgentModeDeep:
		return newDeepAgent(cfg, convSrv, modelFactory, skillRegistry, proxy, memoryStore, ruleEngine)
	case agentconfig.AgentModeReact, "":
		return newReactAgent(cfg, convSrv, modelFactory, skillRegistry, proxy, memoryStore, ruleEngine)
	default:
		return nil, fmt.Errorf("unknown agent mode %q: must be one of %q or %q",
			cfg.AgentMode, agentconfig.AgentModeReact, agentconfig.AgentModeDeep)
	}
}
