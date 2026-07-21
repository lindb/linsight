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

package subagent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	aievent "github.com/lindb/linsight/ai/components/event"
	agentconfig "github.com/lindb/linsight/ai/config"
	"github.com/lindb/linsight/pkg/uuid"
)

// Streamer is the minimal model surface the runner needs: streaming for the ReAct loop and
// non-streaming Generate for step-title naming. *modelcomp.DynamicModelProxy satisfies it
// (it exposes both), so production wiring uses the real per-org model; tests inject a stub.
type Streamer interface {
	Stream(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.StreamReader[*schema.Message], error)
	Generate(ctx context.Context, input []*schema.Message, opts ...einomodel.Option) (*schema.Message, error)
}

// runner is the default Runner implementation: a linear (no-interrupt) ReAct loop that
// reuses the shared streaming/tool primitives and the event lifecycle objects.
type runner struct {
	proxy Streamer
	cfg   *agentconfig.Config
}

// NewRunner builds a Runner backed by proxy and cfg.
func NewRunner(proxy Streamer, cfg *agentconfig.Config) Runner {
	return &runner{proxy: proxy, cfg: cfg}
}

// Run executes the sub-agent ReAct loop and returns the final assistant text.
func (r *runner) Run(ctx context.Context, req RunRequest) (string, error) {
	scope, ok := ScopeFrom(ctx)
	if !ok {
		return "", errors.New("subagent: no scope in context")
	}

	// Depth + cycle guard.
	stack := agentStackFrom(ctx)
	for _, n := range stack {
		if n == req.AgentID {
			return "", fmt.Errorf("subagent: cycle detected, %q already active", req.AgentID)
		}
	}
	if maxDepth := r.cfg.MaxSubAgentDepth; maxDepth > 0 && len(stack) >= maxDepth {
		return "", fmt.Errorf("subagent: max nesting depth %d exceeded", maxDepth)
	}
	ctx = pushAgent(ctx, req.AgentID)

	parentToolCallID := ParentToolCallIDFrom(ctx)
	w := scope.Writer.Scoped(req.AgentID, parentToolCallID)
	cc := scope.ConvCtx.Fork(req.AgentID, parentToolCallID)
	saver := scope.Saver

	tools := r.filterTools(ctx, scope.BaseTools, req.ToolWhitelist)

	msgs := r.initialMessages(ctx, req, scope)

	// A single AssistantTextEvent is reused across iterations (BeginSegment resets per step).
	asst := aievent.NewAssistantTextEvent(w, saver, cc)

	maxIters := r.cfg.MaxStep
	if maxIters <= 0 {
		maxIters = 12
	}

	var lastText string
	for i := 0; i < maxIters; i++ {
		stepID := uuid.GenerateShortUUID()
		stepEvt := aievent.NewStepEvent(w, saver, cc)
		stepEvt.OnStart(ctx, stepID, fmt.Sprintf("Step %d", i+1))

		var modelOpts []einomodel.Option
		if infos := toolInfos(ctx, tools); len(infos) > 0 {
			modelOpts = append(modelOpts, einomodel.WithTools(infos))
		}

		reader, serr := r.proxy.Stream(ctx, msgs, modelOpts...)
		if serr != nil {
			stepEvt.OnFinish(ctx)
			return "", fmt.Errorf("subagent %q stream: %w", req.AgentID, serr)
		}

		asstMsgUID := uuid.GenerateShortUUID()
		asst.BeginSegment(asstMsgUID, stepID)
		reasoning := asst.NewReasoningEvent(stepID)

		content, toolCalls, derr := DrainModelStream(ctx, reader, asst, reasoning)
		reader.Close()
		if derr != nil {
			stepEvt.OnFinish(ctx)
			return "", derr
		}
		lastText = content

		// Under the "llm" strategy, name the step via a concurrent Generate call that overlaps
		// the tool execution below; OnFinish joins it. Same rule as the host agent.
		if r.cfg.LLMStepName() {
			stepEvt.SetNameSource(StartStepTitle(ctx, r.proxy, content, toolCalls))
		}

		// No tool calls: terminal text-only iteration — the final answer.
		if len(toolCalls) == 0 {
			stepEvt.OnFinish(ctx)
			return content, nil
		}

		msgs = append(msgs, &schema.Message{
			Role:      schema.Assistant,
			Content:   content,
			ToolCalls: toolCalls,
		})

		toolMap := invokableMap(ctx, tools)
		results := DispatchToolCalls(toolCalls, r.cfg.MaxParallelToolCalls, func(idx int) *schema.Message {
			return RunToolCall(ctx, &toolCalls[idx], toolMap, tools, w, saver, cc, asstMsgUID, stepID, false)
		})
		msgs = append(msgs, results...)
		stepEvt.OnFinish(ctx)
	}

	subLog.Warn("subagent exceeded step budget",
		logger.String("agent", req.AgentID), logger.Int("max_step", maxIters))
	return lastText, nil
}

// initialMessages seeds the sub-agent conversation per RunMode. The system prompt always
// leads; fork_with_context then appends the inherited (already sanitized) parent history.
func (r *runner) initialMessages(ctx context.Context, req RunRequest, scope *Scope) []*schema.Message {
	msgs := []*schema.Message{{Role: schema.System, Content: req.SystemPrompt}}
	if req.Mode == ModeSkillForkWithContext && scope.History != nil {
		if hist, err := scope.History(ctx); err != nil {
			subLog.Warn("subagent: load history for fork_with_context failed, using blank slate",
				logger.String("agent", req.AgentID), logger.Error(err))
		} else {
			msgs = append(msgs, hist...)
		}
	}
	if strings.TrimSpace(req.Goal) != "" {
		msgs = append(msgs, &schema.Message{Role: schema.User, Content: req.Goal})
	}
	return msgs
}

// filterTools drops approval-gated tools (sub-agents have no human gate) and, when a
// whitelist is supplied, keeps only the named tools. handoff/skill tools survive so
// nesting remains possible (bounded by the depth/cycle guard).
func (r *runner) filterTools(ctx context.Context, base []tool.BaseTool, whitelist []string) []tool.BaseTool {
	wl := make(map[string]bool, len(whitelist))
	for _, n := range whitelist {
		if n = strings.TrimSpace(n); n != "" {
			wl[n] = true
		}
	}
	out := make([]tool.BaseTool, 0, len(base))
	for _, t := range base {
		info, err := t.Info(ctx)
		if err != nil || info == nil {
			continue
		}
		if ToolNeedsApproval(info.Extra) {
			continue
		}
		if len(wl) > 0 && !wl[info.Name] {
			continue
		}
		out = append(out, t)
	}
	return out
}

// toolInfos collects the ToolInfo schemas for the given tools, skipping any that fail.
func toolInfos(ctx context.Context, tools []tool.BaseTool) []*schema.ToolInfo {
	infos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	return infos
}

// invokableMap builds a name → InvokableTool map for O(1) dispatch.
func invokableMap(ctx context.Context, tools []tool.BaseTool) map[string]tool.InvokableTool {
	m := make(map[string]tool.InvokableTool, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		it, ok := t.(tool.InvokableTool)
		if !ok {
			continue
		}
		m[info.Name] = it
	}
	return m
}
