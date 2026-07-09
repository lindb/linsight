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

// Package agent provides the streaming agent runtime built on an explicit
// four-node Lambda graph (thinking → execution → interruptCheck → tools) with
// human-in-the-loop tool approval. It is intentionally separate from ai/service/
// (which handles management CRUD) to keep streaming orchestration concerns isolated.
package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	rulescomp "github.com/lindb/linsight/ai/components/rules"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentservice "github.com/lindb/linsight/ai/service"
)

var agentLog = logger.GetLogger("agent", "runtime")

// Agent orchestrates LLM streaming chat with per-org model routing and AG-UI SSE delivery.
type Agent interface {
	// Chat runs one conversation turn, streaming AG-UI events to writer.
	// convUID identifies an existing conversation; an empty string creates a new one.
	// Returns the conversation UID, the run ID (always set, even on early failure), and any error.
	Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (string, string, error)
}

// agentRuntime is the concrete implementation of the Agent interface. It drives the
// four-node Lambda graph, which handles AG-UI tool-bound interrupts with human-in-the-loop
// approval (see buildGraph in graph.go).
type agentRuntime struct {
	cfg           *agentconfig.Config
	convSrv       agentservice.ConversationService
	modelFactory  *modelcomp.ModelFactory
	skillRegistry *skillscomp.SkillRegistry
	proxy         *modelcomp.DynamicModelProxy
	memoryStore   memory.MemoryStore
	// ruleEngine injects DB-backed global/per-org rules into the LLM context.
	ruleEngine rulescomp.RuleEngine
	// graph is the explicit compose.Graph compiled once at startup.
	graph compose.Runnable[[]*schema.Message, []*schema.Message]
	// sems maps orgID (int64) → chan struct{} (buffered semaphore).
	sems sync.Map
}

// New creates an Agent. The graph is compiled eagerly at construction time using
// context.Background() so that any graph-build failure surfaces at startup rather
// than on the first request.
func New(
	cfg *agentconfig.Config,
	convSrv agentservice.ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
	ruleEngine rulescomp.RuleEngine,
) (Agent, error) {
	graph, err := buildGraph(cfg)
	if err != nil {
		return nil, fmt.Errorf("compile agent graph: %w", err)
	}
	return &agentRuntime{
		cfg:           cfg,
		convSrv:       convSrv,
		modelFactory:  modelFactory,
		skillRegistry: skillRegistry,
		proxy:         proxy,
		memoryStore:   memoryStore,
		ruleEngine:    ruleEngine,
		graph:         graph,
	}, nil
}

// getSemaphore returns (or lazily creates) the per-org concurrency semaphore.
func (a *agentRuntime) getSemaphore(orgID int64) chan struct{} {
	if v, ok := a.sems.Load(orgID); ok {
		return v.(chan struct{})
	}
	sem := make(chan struct{}, a.cfg.MaxConcurrentPerOrg)
	v, _ := a.sems.LoadOrStore(orgID, sem)
	return v.(chan struct{})
}

// Chat runs a full streaming conversation turn and delivers AG-UI events over writer.
func (a *agentRuntime) Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (string, string, error) {
	rt := newChatRun(a, ctx, req, writer, convUID)
	err := rt.run()
	createdUID := ""
	if rt.conv != nil {
		createdUID = rt.conv.UID
	}
	return createdUID, rt.runID, err
}
