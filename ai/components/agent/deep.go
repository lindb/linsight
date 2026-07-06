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
	"errors"
	"fmt"
	"io"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	aievent "github.com/lindb/linsight/ai/components/event"
	"github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentservice "github.com/lindb/linsight/ai/service"
	"github.com/lindb/linsight/pkg/uuid"
)

// deepCtxKey is the key type for deep mode per-request context values.
type deepCtxKey int8

const (
	// deepCtxChatRun injects *chatRun into Lambda node contexts.
	// All per-request state (proxy, tools, events, writer) is accessed through this
	// single key to minimize coupling between the graph nodes and the agent runtime.
	deepCtxChatRun deepCtxKey = 0
)

// deepAgentState is the per-run mutable state threaded through the graph via compose.ProcessState.
// IsFinished signals that the LLM produced "Conclusion: [FINISHED]" and the graph should stop.
// Using a pointer (*deepAgentState) as the type parameter ensures mutations propagate across nodes.
type deepAgentState struct {
	IsFinished bool
}

// createThinkingNode builds the first Lambda in the deep graph.
// It opens a new StepEvent for the current model-call iteration and resets IsFinished
// so a completion signal from a prior tool-less pass does not short-circuit the next cycle.
func createThinkingNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(deepCtxChatRun).(*chatRun)

		// Reset IsFinished for this new iteration so a previous "FINISHED" cannot block
		// future iterations if the graph re-enters thinking after tool execution.
		if err := compose.ProcessState[*deepAgentState](ctx, func(_ context.Context, s *deepAgentState) error {
			s.IsFinished = false
			return nil
		}); err != nil {
			return nil, fmt.Errorf("thinking: reset state: %w", err)
		}

		// Open a new step to group the upcoming model call and tool executions.
		stepID := uuid.GenerateShortUUID()
		stepEvt := aievent.NewStepEvent(rt.writer, rt.agent.convSrv, rt.convCtx)
		stepEvt.OnStart(ctx, stepID, "Deep Analysis")
		rt.currentStepID = stepID
		rt.currentStepEvt = stepEvt

		return messages, nil
	})
}

// createExecutionNode builds the LLM-call Lambda.
// It streams reasoning and text tokens to the SSE writer, accumulates tool_call fragments
// (OpenAI streaming protocol), and returns the updated message list.
// If no tool calls are produced, the step is closed here (text-only iteration).
func createExecutionNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(deepCtxChatRun).(*chatRun)

		// Collect tool schemas for per-request binding so the LLM knows available tools.
		toolInfos := make([]*schema.ToolInfo, 0, len(rt.orgTools))
		for _, t := range rt.orgTools {
			info, err := t.Info(ctx)
			if err != nil {
				agentLog.Warn("deep: failed to get tool info, skipping", logger.Error(err))
				continue
			}
			toolInfos = append(toolInfos, info)
		}

		var modelOpts []einomodel.Option
		if len(toolInfos) > 0 {
			modelOpts = append(modelOpts, einomodel.WithTools(toolInfos))
		}

		reader, err := rt.agent.proxy.Stream(ctx, messages, modelOpts...)
		if err != nil {
			return nil, fmt.Errorf("deep execution stream: %w", err)
		}
		defer reader.Close()

		reasoningEvt := rt.assistantEvt.NewReasoningEvent(rt.currentStepID)

		// toolCallsByIndex accumulates partial ToolCall fragments using the OpenAI streaming
		// index as key; toolCallsIndexOrder preserves insertion order for deterministic merging.
		toolCallsByIndex := make(map[int]*schema.ToolCall)
		var toolCallsIndexOrder []int
		var contentSB strings.Builder

		for {
			chunk, recvErr := reader.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				return nil, fmt.Errorf("deep execution recv: %w", recvErr)
			}
			if chunk == nil {
				continue
			}

			if chunk.ReasoningContent != "" {
				reasoningEvt.OnContent(chunk.ReasoningContent)
			}
			if chunk.Content != "" {
				rt.assistantEvt.OnContent(chunk.Content)
				contentSB.WriteString(chunk.Content)
			}

			// Merge streaming tool_call fragments: each chunk carries a partial Arguments
			// string that must be concatenated in Index order.
			for i := range chunk.ToolCalls {
				tc := &chunk.ToolCalls[i]
				idx := i
				if tc.Index != nil {
					idx = *tc.Index
				}
				if existing, ok := toolCallsByIndex[idx]; ok {
					existing.Function.Arguments += tc.Function.Arguments
					if tc.ID != "" {
						existing.ID = tc.ID
					}
					if tc.Function.Name != "" {
						existing.Function.Name = tc.Function.Name
					}
					if tc.Type != "" {
						existing.Type = tc.Type
					}
				} else {
					copied := *tc
					toolCallsByIndex[idx] = &copied
					toolCallsIndexOrder = append(toolCallsIndexOrder, idx)
				}
			}
		}

		// Close reasoning and text SSE frames for this model call.
		reasoningEvt.OnEnd(ctx)
		rt.assistantEvt.OnEnd(ctx)

		// Detect explicit completion signal; set state so decideBranch can terminate the loop.
		if strings.Contains(contentSB.String(), "Conclusion: [FINISHED]") {
			if err := compose.ProcessState[*deepAgentState](ctx, func(_ context.Context, s *deepAgentState) error {
				s.IsFinished = true
				return nil
			}); err != nil {
				return nil, fmt.Errorf("deep execution: set finished state: %w", err)
			}
		}

		// Tool calls present: append the assistant message and let the tools node execute.
		// Include Content so the LLM can reference its own [DEEP THINKING] preamble text
		// in subsequent iterations (relevant for non-native-reasoning models).
		if len(toolCallsByIndex) > 0 {
			toolCalls := make([]schema.ToolCall, 0, len(toolCallsByIndex))
			for _, idx := range toolCallsIndexOrder {
				toolCalls = append(toolCalls, *toolCallsByIndex[idx])
			}
			return append(messages, &schema.Message{
				Role:      schema.Assistant,
				Content:   contentSB.String(),
				ToolCalls: toolCalls,
			}), nil
		}

		// No tool calls: this is a text-only iteration; close the step here.
		rt.currentStepEvt.OnFinish(ctx, rt.currentStepID)
		return messages, nil
	})
}

// createToolsNode builds the tool-execution Lambda.
// It runs each tool call from the last assistant message, emits TOOL_CALL_* SSE events,
// appends tool result messages, and closes the current step when done.
func createToolsNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(deepCtxChatRun).(*chatRun)

		if len(messages) == 0 {
			return messages, nil
		}

		// Build a name → InvokableTool map for O(1) dispatch.
		toolMap := make(map[string]tool.InvokableTool, len(rt.orgTools))
		for _, t := range rt.orgTools {
			info, err := t.Info(ctx)
			if err != nil {
				continue
			}
			it, ok := t.(tool.InvokableTool)
			if !ok {
				continue
			}
			toolMap[info.Name] = it
		}

		lastMsg := messages[len(messages)-1]
		for _, tc := range lastMsg.ToolCalls {
			toolEvt := aievent.NewToolCallEvent(
				tc.ID, tc.Function.Name,
				rt.assistantMsgUID, rt.currentStepID,
				rt.writer, rt.agent.convSrv, rt.convCtx,
			)
			toolEvt.OnStart(ctx, tc.Function.Arguments)

			it, ok := toolMap[tc.Function.Name]
			if !ok {
				toolErr := fmt.Errorf("tool %q not found", tc.Function.Name)
				toolEvt.OnError(ctx, toolErr)
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    toolErr.Error(),
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
				})
				continue
			}

			result, runErr := it.InvokableRun(ctx, tc.Function.Arguments)
			if runErr != nil {
				toolEvt.OnError(ctx, runErr)
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    runErr.Error(),
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
				})
			} else {
				toolEvt.OnEnd(ctx, result, nil)
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    result,
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
				})
			}
		}

		// All tools in this iteration are done; close the step.
		rt.currentStepEvt.OnFinish(ctx, rt.currentStepID)
		return messages, nil
	})
}

// decideBranch routes to "tools" if the last message carries tool calls, or to
// compose.END when the model has finished (IsFinished=true) or produced a text-only response.
func decideBranch(ctx context.Context, msgs []*schema.Message) (string, error) {
	var isFinished bool
	if err := compose.ProcessState[*deepAgentState](ctx, func(_ context.Context, s *deepAgentState) error {
		isFinished = s.IsFinished
		return nil
	}); err != nil {
		return "", fmt.Errorf("decideBranch: read state: %w", err)
	}
	if isFinished {
		return compose.END, nil
	}
	if len(msgs) == 0 {
		return compose.END, nil
	}
	lastMsg := msgs[len(msgs)-1]
	if lastMsg.Role == schema.Assistant && len(lastMsg.ToolCalls) > 0 {
		return "tools", nil
	}
	return compose.END, nil
}

// buildDeepGraph compiles the explicit three-node Lambda graph for deep mode.
// The graph topology is: START → thinking → execution → [branch] → tools → thinking (loop).
// MaxRunSteps is multiplied by 3 because each full iteration consumes three steps.
func buildDeepGraph(cfg *agentconfig.Config) (compose.Runnable[[]*schema.Message, []*schema.Message], error) {
	g := compose.NewGraph[[]*schema.Message, []*schema.Message](
		compose.WithGenLocalState(func(_ context.Context) *deepAgentState {
			return &deepAgentState{}
		}),
	)

	if err := g.AddLambdaNode("thinking", createThinkingNode()); err != nil {
		return nil, fmt.Errorf("add thinking node: %w", err)
	}
	if err := g.AddLambdaNode("execution", createExecutionNode()); err != nil {
		return nil, fmt.Errorf("add execution node: %w", err)
	}
	if err := g.AddLambdaNode("tools", createToolsNode()); err != nil {
		return nil, fmt.Errorf("add tools node: %w", err)
	}

	if err := g.AddEdge(compose.START, "thinking"); err != nil {
		return nil, fmt.Errorf("add edge START→thinking: %w", err)
	}
	if err := g.AddEdge("thinking", "execution"); err != nil {
		return nil, fmt.Errorf("add edge thinking→execution: %w", err)
	}

	branch := compose.NewGraphBranch(decideBranch, map[string]bool{
		"tools":     true,
		compose.END: true,
	})
	if err := g.AddBranch("execution", branch); err != nil {
		return nil, fmt.Errorf("add branch execution→tools/END: %w", err)
	}

	if err := g.AddEdge("tools", "thinking"); err != nil {
		return nil, fmt.Errorf("add edge tools→thinking: %w", err)
	}

	maxSteps := cfg.MaxStep * 3
	if maxSteps <= 0 {
		maxSteps = 36 // defensive default: 12 iterations × 3 nodes
	}
	return g.Compile(context.Background(), compose.WithMaxRunSteps(maxSteps))
}

// newDeepAgent creates an agentRuntime configured for deep mode.
// The deep graph is compiled once at startup; per-request state is injected via context.
func newDeepAgent(
	cfg *agentconfig.Config,
	convSrv agentservice.ConversationService,
	modelFactory *modelcomp.ModelFactory,
	skillRegistry *skillscomp.SkillRegistry,
	proxy *modelcomp.DynamicModelProxy,
	memoryStore memory.MemoryStore,
) (Agent, error) {
	deepGraph, err := buildDeepGraph(cfg)
	if err != nil {
		return nil, fmt.Errorf("compile deep graph: %w", err)
	}
	return &agentRuntime{
		cfg:           cfg,
		convSrv:       convSrv,
		modelFactory:  modelFactory,
		skillRegistry: skillRegistry,
		proxy:         proxy,
		memoryStore:   memoryStore,
		deepGraph:     deepGraph,
		agentMode:     agentconfig.AgentModeDeep,
	}, nil
}
