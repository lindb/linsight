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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/agui"
	aievent "github.com/lindb/linsight/ai/components/event"
	toolscomp "github.com/lindb/linsight/ai/components/tools"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/uuid"
)

// ctxKey is the key type for per-request context values.
type ctxKey int8

const (
	// ctxChatRun injects *chatRun into Lambda node contexts.
	// All per-request state (proxy, tools, events, writer) is accessed through this
	// single key to minimize coupling between the graph nodes and the agent runtime.
	ctxChatRun ctxKey = 0
)

// agentState is the per-run mutable state threaded through the graph via compose.ProcessState.
// Using a pointer (*agentState) as the type parameter ensures mutations propagate across nodes.
type agentState struct {
	// IsFinished is true when the LLM produced "Conclusion: [FINISHED]" (RCA path) or a
	// text-only response with no tool calls; decideBranch then terminates the graph.
	IsFinished bool
	// GoToThinking is set by the rejection branch (processResumeDecision) so decideBranch
	// routes back to "thinking" for another LLM call instead of going to END.
	GoToThinking bool
	// InterruptIssued is set when processInterrupt fires so decideBranch routes to END
	// (the run terminates immediately; the client must send a resume request).
	InterruptIssued bool
}

// toolNeedsApproval checks ToolInfo.Extra for _gov_need_approval=true.
// Returns false if extra is nil, the key is absent, or the value is not a bool.
func toolNeedsApproval(extra map[string]any) bool {
	if extra == nil {
		return false
	}
	v, ok := extra["_gov_need_approval"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

// getToolPublicExtra returns a shallow copy of extra with all _gov_* keys stripped.
// This prevents governance metadata from leaking to SSE frames or DB rows visible
// to the frontend. Returns nil if the result would be empty.
func getToolPublicExtra(extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return extra
	}
	result := make(map[string]any, len(extra))
	for k, v := range extra {
		if !strings.HasPrefix(k, "_gov_") {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// parseToolResult unwraps a ToolCallResult envelope (if present) and injects
// raw_data into extra for rich frontend rendering (e.g. DataQueryView table).
func parseToolResult(result string) (llmContent string, extra map[string]any) {
	if tcr, ok := toolscomp.ParseToolCallResult(result); ok {
		llmContent = string(tcr.Data)
		extra = tcr.Extra
		if extra == nil {
			extra = make(map[string]any)
		}
		var rawData json.RawMessage
		if json.Unmarshal(tcr.Data, &rawData) == nil {
			extra["raw_data"] = rawData
		}
		return llmContent, extra
	}
	return result, nil
}

// createThinkingNode builds the first Lambda in the graph.
// It opens a new StepEvent for the current model-call iteration and resets IsFinished
// so a completion signal from a prior tool-less pass does not short-circuit the next cycle.
func createThinkingNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(ctxChatRun).(*chatRun)

		// Reset per-iteration flags so stale values from a previous loop cannot
		// short-circuit or repeat special-case logic (e.g. a prior "FINISHED", or the
		// GoToThinking set by a rejection branch that already looped back here).
		if err := compose.ProcessState[*agentState](ctx, func(_ context.Context, s *agentState) error {
			s.IsFinished = false
			s.GoToThinking = false
			s.InterruptIssued = false
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
		rt := ctx.Value(ctxChatRun).(*chatRun)

		// Resume first pass: skip the LLM entirely. On a resume run the ToolCalls were
		// already produced by the LLM in run 1 and are reconstructed from pendingInterrupt
		// by the interruptCheck node; calling the LLM again would discard the user's
		// approval decision. Subsequent thinking iterations (e.g. after a rejection loops
		// back) fall through and call the LLM normally.
		if rt.isResume && !rt.resumeConsumed {
			rt.resumeConsumed = true
			return messages, nil
		}

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

		// Dynamic rule injection: match DB-backed rules against the tools invoked so far
		// and the conversation text, then splice the compiled rule text into a COPY of the
		// system message. injectRulesIntoSystem never mutates the thread-state `messages`
		// slice, so multi-iteration loops do not stack rule blocks on top of each other.
		msgsForLLM := messages
		if rt.agent.ruleEngine != nil {
			rulesText, rerr := rt.agent.ruleEngine.MatchAndCompile(ctx,
				&einomodel.CallbackInput{Messages: messages, Tools: toolInfos})
			if rerr != nil {
				agentLog.Warn("deep: rule match failed, proceeding without rules", logger.Error(rerr))
			} else if rulesText != "" {
				msgsForLLM = injectRulesIntoSystem(messages, rulesText)
			}
		}

		// Inject org-level long-term memory into the system prompt (copy-on-write).
		// Follows the same pattern as rule injection so multi-iteration loops do not
		// stack memory blocks.
		if rt.agent.memoryBackend != nil {
			memText, merr := rt.agent.memoryBackend.Render(ctx)
			if merr != nil {
				agentLog.Warn("deep: memory render failed, proceeding without memory", logger.Error(merr))
			} else if memText != "" {
				msgsForLLM = injectMemoryIntoSystem(msgsForLLM, memText)
			}
		}

		reader, err := rt.agent.proxy.Stream(ctx, msgsForLLM, modelOpts...)
		if err != nil {
			return nil, fmt.Errorf("deep execution stream: %w", err)
		}
		defer reader.Close()

		// Open a fresh text segment for this iteration: its own AG-UI message ID, timer,
		// and DB row. currentAssistantMsgUID is the parentMessageId for any tool calls this
		// step produces (both live SSE and history replay).
		stepAsstMsgUID := uuid.GenerateShortUUID()
		rt.currentAssistantMsgUID = stepAsstMsgUID
		rt.assistantEvt.BeginSegment(stepAsstMsgUID, rt.currentStepID)

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
			if err := compose.ProcessState[*agentState](ctx, func(_ context.Context, s *agentState) error {
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
		rt.currentStepEvt.OnFinish(ctx)
		return messages, nil
	})
}

// processInterrupt handles the interrupt case: one or more tool calls need approval and
// AutoApprove is false. It maps to the AG-UI timeline moment "agent pauses, emits
// RUN_FINISHED(outcome=interrupt)". For each pending tool call it emits TOOL_CALL_START
// SSE and persists the tool_call DB row (so the frontend can render the approval card),
// then persists the assistant message, closes the step, saves an InterruptRecord, and
// sets InterruptIssued=true so decideBranch routes to END.
func processInterrupt(ctx context.Context, messages []*schema.Message, rt *chatRun) ([]*schema.Message, error) {
	lastMsg := messages[len(messages)-1]
	pendingCalls := make([]agui.PendingToolCall, 0, len(lastMsg.ToolCalls))

	for _, tc := range lastMsg.ToolCalls {
		rawExtra := cloneToolExtra(rt.orgTools, tc.Function.Name)
		publicExtra := getToolPublicExtra(rawExtra)

		// Emit TOOL_CALL_START SSE + persist tool_call DB row so the frontend can
		// render the approval card with the correct tool name and arguments.
		toolEvt := aievent.NewToolCallEvent(
			tc.ID, tc.Function.Name,
			rt.currentAssistantMsgUID, rt.currentStepID,
			rt.writer, rt.agent.convSrv, rt.convCtx,
		)
		toolEvt.OnStart(ctx, tc.Function.Arguments, publicExtra)

		var cardType string
		if rawExtra != nil {
			if ct, ok := rawExtra["_gov_card_type"]; ok {
				cardType, _ = ct.(string)
			}
		}
		pendingCalls = append(pendingCalls, agui.PendingToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
			CardType:  cardType,
		})
	}

	// The interrupted step's text segment was already persisted by the execution node's
	// OnEnd (which runs before interruptCheck). Just close the step so the tool_call rows
	// written above always have a parent in the event table.
	rt.currentStepEvt.OnFinish(ctx)

	// Serialize the full []schema.ToolCall for later resume (args may be edited by user).
	pendingJSON, err := json.Marshal(lastMsg.ToolCalls)
	if err != nil {
		return nil, fmt.Errorf("interrupt: marshal tool calls: %w", err)
	}

	interruptUID := uuid.GenerateShortUUID()
	payload := &agui.InterruptPayload{
		InterruptID: interruptUID,
		ToolCalls:   pendingCalls,
	}
	// Persist the client-facing payload verbatim (including per-tool CardType, which is
	// stripped from the events table) so history replay can reproduce the interrupt frame
	// without access to the tool registry. Status is refreshed from the record on replay.
	clientPayloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("interrupt: marshal client payload: %w", err)
	}

	rec := &agentmodel.InterruptRecord{
		UID:              interruptUID,
		ConvUID:          rt.conv.UID,
		ConvID:           rt.conv.ID,
		OrgID:            rt.orgID,
		UserID:           rt.userID,
		OrigRunID:        rt.runID,
		Status:           agentmodel.InterruptStatusPending,
		PendingToolCalls: json.RawMessage(pendingJSON),
		ClientPayload:    json.RawMessage(clientPayloadJSON),
		AssistantMsgUID:  rt.currentAssistantMsgUID,
		StepID:           rt.currentStepID,
	}
	rec.CreatedBy = rt.userID
	rec.UpdatedBy = rt.userID
	if err := rt.agent.convSrv.SaveInterruptRecord(ctx, rec); err != nil {
		return nil, fmt.Errorf("interrupt: save record: %w", err)
	}

	rt.interruptPayload = payload
	rt.interruptHandled = true

	if err := compose.ProcessState[*agentState](ctx, func(_ context.Context, s *agentState) error {
		s.InterruptIssued = true
		return nil
	}); err != nil {
		return nil, fmt.Errorf("interrupt: set state: %w", err)
	}
	return messages, nil
}

// processResumeDecision handles a resume request. It maps to the AG-UI timeline moment
// "client sends resume payload with approved/editedArgs".
//   - Rejection: append a rejection tool message per pending TC, emit TOOL_CALL_RESULT(error),
//     close the step, mark the record rejected, and set GoToThinking so the graph loops
//     back to the LLM for an apology/alternative.
//   - Approval: apply optional editedArgs overrides and stage the resolved []schema.ToolCall
//     in rt.pendingToolCallsToExecute for the tools node.
func processResumeDecision(ctx context.Context, messages []*schema.Message, rt *chatRun) ([]*schema.Message, error) {
	rt.resumeHandled = true
	ir := rt.pendingInterrupt

	var pendingTCs []schema.ToolCall
	if err := json.Unmarshal(ir.PendingToolCalls, &pendingTCs); err != nil {
		return nil, fmt.Errorf("resume: unmarshal pending tool calls: %w", err)
	}

	resume := rt.req.Resume

	if !resume.Approved {
		// Rejection: emit error TOOL_CALL_RESULT for each TC (no OnStart — run 1's
		// processInterrupt already emitted TOOL_CALL_START + persisted the tool_call DB row).
		rejectionErr := errors.New("user rejected the tool invocation")
		for _, tc := range pendingTCs {
			toolEvt := aievent.NewToolCallEvent(
				tc.ID, tc.Function.Name,
				ir.AssistantMsgUID, ir.StepID,
				rt.writer, rt.agent.convSrv, rt.convCtx,
			)
			toolEvt.OnError(ctx, rejectionErr)
			messages = append(messages, &schema.Message{
				Role:       schema.Tool,
				Content:    rejectionErr.Error(),
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
			})
		}
		rt.currentStepEvt.OnFinish(ctx)

		if err := rt.agent.convSrv.UpdateInterruptStatus(ctx, ir.UID,
			agentmodel.InterruptStatusRejected, resume.Comment); err != nil {
			agentLog.Warn("resume: failed to update interrupt status to rejected", logger.Error(err))
		}

		if err := compose.ProcessState[*agentState](ctx, func(_ context.Context, s *agentState) error {
			s.GoToThinking = true
			return nil
		}); err != nil {
			return nil, fmt.Errorf("resume: set GoToThinking: %w", err)
		}
		return messages, nil
	}

	// Approval: apply any per-tool argument overrides the user submitted.
	toolCalls := make([]schema.ToolCall, len(pendingTCs))
	copy(toolCalls, pendingTCs)
	if len(resume.EditedArgs) > 0 {
		for i := range toolCalls {
			if overrideRaw, ok := resume.EditedArgs[toolCalls[i].ID]; ok {
				overrideBytes, err := json.Marshal(overrideRaw)
				if err != nil {
					return nil, fmt.Errorf("resume: marshal edited args for %s: %w", toolCalls[i].ID, err)
				}
				toolCalls[i].Function.Arguments = string(overrideBytes)
			}
		}
	}
	rt.pendingToolCallsToExecute = toolCalls

	if err := rt.agent.convSrv.UpdateInterruptStatus(ctx, ir.UID,
		agentmodel.InterruptStatusApproved, resume.Comment); err != nil {
		agentLog.Warn("resume: failed to update interrupt status to approved", logger.Error(err))
	}
	return messages, nil
}

// createInterruptCheckNode implements the four-case governance gate that sits between
// the execution (LLM) node and the tools node. Each case maps to an AG-UI timeline moment:
//
//   - Case A (text-only / FINISHED): no ToolCalls on last message → pass through → decideBranch ends the run
//   - Case B (resume): pending interrupt from a previous run → processResumeDecision (approve/reject)
//   - Case C (normal, no approval needed): ToolCalls present and none flagged _gov_need_approval → pass to tools
//   - Case D (interrupt): ToolCalls present, at least one _gov_need_approval=true, AutoApprove=false → processInterrupt
func createInterruptCheckNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(ctxChatRun).(*chatRun)

		// Case A: LLM produced a text-only response — nothing to gate.
		if len(messages) == 0 {
			return messages, nil
		}
		lastMsg := messages[len(messages)-1]
		if lastMsg.Role != schema.Assistant || len(lastMsg.ToolCalls) == 0 {
			return messages, nil
		}

		// Case B: this is a resume run and we have not yet consumed the interrupt decision.
		if rt.isResume && !rt.resumeHandled && rt.pendingInterrupt != nil {
			return processResumeDecision(ctx, messages, rt)
		}

		// Cases C & D: fresh ToolCalls. If ANY tool needs approval and AutoApprove is
		// false, trigger the interrupt (Case D); otherwise fall through to tools (Case C).
		if !rt.req.AutoApprove {
			for _, tc := range lastMsg.ToolCalls {
				rawExtra := cloneToolExtra(rt.orgTools, tc.Function.Name)
				if toolNeedsApproval(rawExtra) {
					return processInterrupt(ctx, messages, rt)
				}
			}
		}

		// Case C: all tools are pre-approved; let the tools node execute normally.
		return messages, nil
	})
}

// createToolsNode builds the tool-execution Lambda.
// It runs each tool call from the last assistant message, emits TOOL_CALL_* SSE events,
// appends tool result messages, and closes the current step when done.
//
// On an approved resume run the ToolCalls come from rt.pendingToolCallsToExecute (possibly
// carrying user-edited arguments) and OnStart is skipped, because run 1's processInterrupt
// already emitted TOOL_CALL_START + persisted the tool_call DB row.
func createToolsNode() *compose.Lambda {
	return compose.InvokableLambda(func(ctx context.Context, messages []*schema.Message) ([]*schema.Message, error) {
		rt := ctx.Value(ctxChatRun).(*chatRun)

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

		// Determine which tool calls to execute and whether to skip OnStart.
		// Approved resume: use pendingToolCallsToExecute (args may have been edited) and
		// skip OnStart because run 1's processInterrupt already emitted TOOL_CALL_START.
		isResumeApproved := rt.isResume && len(rt.pendingToolCallsToExecute) > 0
		var toolCalls []schema.ToolCall
		if isResumeApproved {
			toolCalls = rt.pendingToolCallsToExecute
		} else {
			lastMsg := messages[len(messages)-1]
			toolCalls = lastMsg.ToolCalls
		}

		// parentMsgUID / stepID link tool_result events to their parent assistant message
		// and step. On resume, reuse the UIDs from the original interrupted run.
		parentMsgUID := rt.currentAssistantMsgUID
		stepID := rt.currentStepID
		if isResumeApproved && rt.pendingInterrupt != nil {
			parentMsgUID = rt.pendingInterrupt.AssistantMsgUID
			stepID = rt.pendingInterrupt.StepID
		}

		for _, tc := range toolCalls {
			toolEvt := aievent.NewToolCallEvent(
				tc.ID, tc.Function.Name,
				parentMsgUID, stepID,
				rt.writer, rt.agent.convSrv, rt.convCtx,
			)

			if !isResumeApproved {
				// Normal execution path: emit TOOL_CALL_START + persist tool_call DB row.
				staticExtra := cloneToolExtra(rt.orgTools, tc.Function.Name)
				publicExtra := getToolPublicExtra(staticExtra)
				toolEvt.OnStart(ctx, tc.Function.Arguments, publicExtra)
			}
			// Resume+approved: run 1's processInterrupt already called OnStart.

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
				llmContent, extra := parseToolResult(result)
				toolEvt.OnEnd(ctx, llmContent, extra)
				messages = append(messages, &schema.Message{
					Role:       schema.Tool,
					Content:    llmContent,
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
				})
			}
		}

		// All tools in this iteration are done; close the step.
		rt.currentStepEvt.OnFinish(ctx)

		// Clear the resume tool calls so subsequent loop iterations use the normal path.
		rt.pendingToolCallsToExecute = nil
		return messages, nil
	})
}

// decideBranch routes after the interruptCheck node:
//   - compose.END  when InterruptIssued=true (run halted for human approval)
//   - compose.END  when IsFinished=true (RCA "FINISHED" or text-only response)
//   - "thinking"   when GoToThinking=true (rejection → loop back to the LLM)
//   - "tools"      when rt.pendingToolCallsToExecute is non-empty (approved resume)
//   - "tools"      when the last message carries ToolCalls (normal execution, Case C)
//   - compose.END  otherwise
func decideBranch(ctx context.Context, msgs []*schema.Message) (string, error) {
	var state agentState
	if err := compose.ProcessState[*agentState](ctx, func(_ context.Context, s *agentState) error {
		state = *s
		return nil
	}); err != nil {
		return "", fmt.Errorf("decideBranch: read state: %w", err)
	}

	if state.InterruptIssued {
		return compose.END, nil
	}
	if state.IsFinished {
		return compose.END, nil
	}
	if state.GoToThinking {
		return "thinking", nil
	}

	rt := ctx.Value(ctxChatRun).(*chatRun)
	if len(rt.pendingToolCallsToExecute) > 0 {
		return "tools", nil
	}

	if len(msgs) > 0 {
		lastMsg := msgs[len(msgs)-1]
		if lastMsg.Role == schema.Assistant && len(lastMsg.ToolCalls) > 0 {
			return "tools", nil
		}
	}
	return compose.END, nil
}

// buildGraph compiles the explicit four-node Lambda graph.
// The interruptCheck node implements AG-UI tool-bound interrupts (human-in-the-loop approval).
// MaxRunSteps is multiplied by 4 because each full iteration consumes four steps.
//
// Graph topology (nodes = Lambda; [branch] = decideBranch conditional edge):
//
//	          ┌─────────────────────────── tools ◀────────────┐ (Case C / approved resume)
//	          │                              │                 │
//	          ▼                              ▼                 │
//	START → thinking → execution → interruptCheck → [branch] ──┤
//	          ▲                                                 │
//	          └──────────────── thinking ◀──────────────────── │ (GoToThinking: rejected resume)
//	                                                            │
//	                                                    END ◀───┘ (IsFinished / InterruptIssued)
//
// Node roles and their AG-UI timeline mapping:
//
//	thinking       — opens a STEP; resets per-iteration state flags.
//	execution      — streams the LLM (REASONING_* / TEXT_MESSAGE_*); on a resume first
//	                 pass it skips the LLM and reuses the ToolCalls from the interrupt.
//	interruptCheck — the governance gate, one branch per AG-UI moment:
//	                   Case A  text-only / "Conclusion: [FINISHED]"  → pass through → END
//	                   Case B  resume payload (approve / reject)      → processResumeDecision
//	                   Case C  tool calls, none need approval         → pass through → tools
//	                   Case D  tool call needs approval, no auto-ok   → processInterrupt
//	                           (emits TOOL_CALL_START + RUN_FINISHED[outcome=interrupt], halts)
//	tools          — executes tool calls (TOOL_CALL_RESULT), then loops back to thinking.
//
// decideBranch reads agentState after interruptCheck and routes to one of:
//
//	END       InterruptIssued=true (Case D) | IsFinished=true (Case A)
//	thinking  GoToThinking=true (rejected resume → LLM apologises / re-plans)
//	tools     approved resume (pendingToolCallsToExecute) | Case C (fresh tool calls)
func buildGraph(cfg *agentconfig.Config) (compose.Runnable[[]*schema.Message, []*schema.Message], error) {
	g := compose.NewGraph[[]*schema.Message, []*schema.Message](
		compose.WithGenLocalState(func(_ context.Context) *agentState {
			return &agentState{}
		}),
	)

	if err := g.AddLambdaNode("thinking", createThinkingNode()); err != nil {
		return nil, fmt.Errorf("add thinking node: %w", err)
	}
	if err := g.AddLambdaNode("execution", createExecutionNode()); err != nil {
		return nil, fmt.Errorf("add execution node: %w", err)
	}
	if err := g.AddLambdaNode("interruptCheck", createInterruptCheckNode()); err != nil {
		return nil, fmt.Errorf("add interruptCheck node: %w", err)
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
	if err := g.AddEdge("execution", "interruptCheck"); err != nil {
		return nil, fmt.Errorf("add edge execution→interruptCheck: %w", err)
	}

	branch := compose.NewGraphBranch(decideBranch, map[string]bool{
		"tools":     true,
		"thinking":  true,
		compose.END: true,
	})
	if err := g.AddBranch("interruptCheck", branch); err != nil {
		return nil, fmt.Errorf("add branch interruptCheck→tools/thinking/END: %w", err)
	}

	if err := g.AddEdge("tools", "thinking"); err != nil {
		return nil, fmt.Errorf("add edge tools→thinking: %w", err)
	}

	maxSteps := cfg.MaxStep * 4
	if maxSteps <= 0 {
		maxSteps = 48 // defensive default: 12 iterations × 4 nodes
	}
	return g.Compile(context.Background(), compose.WithMaxRunSteps(maxSteps))
}

// cloneToolExtra returns a shallow copy of the named tool's ToolInfo.Extra,
// or nil if the tool is not found or has no Extra. Used to attach display metadata
// (icon, render_type) to tool events without coupling the graph nodes to specific
// tool implementations.
func cloneToolExtra(tools []tool.BaseTool, name string) map[string]any {
	for _, t := range tools {
		info, err := t.Info(context.Background())
		if err != nil || info == nil || info.Name != name {
			continue
		}
		if len(info.Extra) == 0 {
			return nil
		}
		m := make(map[string]any, len(info.Extra))
		for k, v := range info.Extra {
			m[k] = v
		}
		return m
	}
	return nil
}
