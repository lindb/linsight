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

package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentservice "github.com/lindb/linsight/ai/service"
)

var agentLog = logger.GetLogger("agent", "runtime")

// agentRuntime is the concrete implementation of the Agent interface.
// It is shared by both "react" and "deep" modes; the agentMode field
// controls which system prompt is injected (see buildSystemPrompt in context.go).
type agentRuntime struct {
	cfg           *agentconfig.Config
	convSrv       agentservice.ConversationService
	modelFactory  *modelcomp.ModelFactory
	skillRegistry *skillscomp.SkillRegistry
	proxy         *modelcomp.DynamicModelProxy
	memoryStore   memory.MemoryStore
	// compiled is the singleton ReAct agent graph built once at startup.
	compiled *react.Agent
	// deepGraph is the explicit compose.Graph used in AgentModeDeep.
	deepGraph compose.Runnable[[]*schema.Message, []*schema.Message]
	// sems maps orgID (int64) → chan struct{} (buffered semaphore).
	sems sync.Map
	// agentMode distinguishes "react" from "deep" at runtime.
	agentMode agentconfig.AgentMode
}

// newReactAgent builds an agentRuntime configured for standard ReAct mode.
func newReactAgent(
	cfg *agentconfig.Config,
	convSrv agentservice.ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
) (Agent, error) {
	compiled, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel:      proxy,
		MaxStep:               cfg.MaxStep,
		StreamToolCallChecker: fullResponseStreamToolCallChecker,
		MessageModifier:       buildLanguageAlignModifier(),
		ToolsConfig: compose.ToolsNodeConfig{
			// When the LLM hallucinates a non-existent tool name, return a descriptive
			// error message so the LLM can self-correct instead of crashing the run.
			UnknownToolsHandler: func(_ context.Context, name, _ string) (string, error) {
				return fmt.Sprintf("tool %q not found; available tools are listed in the system prompt", name), nil
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("compile react graph: %w", err)
	}
	return &agentRuntime{
		cfg:           cfg,
		convSrv:       convSrv,
		modelFactory:  modelFactory,
		skillRegistry: skillRegistry,
		proxy:         proxy,
		memoryStore:   memoryStore,
		compiled:      compiled,
		agentMode:     agentconfig.AgentModeReact,
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

// Chat runs a full ReAct streaming conversation turn and delivers AG-UI events over writer.
func (a *agentRuntime) Chat(ctx context.Context, req *agentapi.ChatRequest, writer agentapi.SSEWriter, convUID string) (string, string, error) {
	rt := newChatRun(a, ctx, req, writer, convUID)
	err := rt.run()
	createdUID := ""
	if rt.conv != nil {
		createdUID = rt.conv.UID
	}
	return createdUID, rt.runID, err
}
