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
	"io"
	"sync/atomic"

	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"

	aievent "github.com/lindb/linsight/ai/components/event"
	"github.com/lindb/linsight/pkg/uuid"
)

// newSSECallback constructs the Eino callbacks.Handler that:
//   - relays model token events as AG-UI SSE frames (TEXT_MESSAGE_*, REASONING_MESSAGE_*)
//   - relays tool invocation events as AG-UI SSE frames (STEP_*, TOOL_CALL_*)
//   - persists all event rows to the DB via the event lifecycle objects on chatRun
func newSSECallback(rt *chatRun) callbacks.Handler {
	toolHandler := &cbtemplate.ToolCallbackHandler{
		// OnStart fires when Eino is about to invoke a tool.
		// Increments pendingToolCalls so OnFinish knows how many tools are outstanding.
		// Creates a ToolCallEvent, caches it by callID, and delegates to its OnStart.
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			// Increment before creating ToolCallEvent so counter is consistent when OnEnd fires.
			atomic.AddInt32(&rt.pendingToolCalls, 1)
			// compose.GetToolCallID reads the call_id injected by Eino before the
			// callback fires, replacing the previous sync.Map lookup (race-prone).
			callID := compose.GetToolCallID(ctx)
			toolEvt := aievent.NewToolCallEvent(callID, info.Name, rt.assistantMsgUID, rt.currentStepID, rt.writer, rt.agent.convSrv, rt.convCtx)
			rt.toolEvtsMu.Lock()
			rt.toolEvts[callID] = toolEvt
			rt.toolEvtsMu.Unlock()
			// Clone static tool metadata (icon/render_type) from ToolInfo.Extra for
			// frontend pre-rendering before the result arrives.
			staticExtra := cloneToolExtra(rt.orgTools, info.Name)
			toolEvt.OnStart(ctx, args, staticExtra)
			return ctx
		},

		// OnEnd fires when a tool returns successfully.
		// Delegates to the cached ToolCallEvent.OnEnd with structured guidance
		// so the frontend can render it with a rich component (table, list, …).
		// Fires STEP_FINISHED when this is the last pending tool in the iteration.
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			callID := compose.GetToolCallID(ctx)

			// Build extra: use runtime extra from ToolCallResult.Extra (if available),
			// then inject raw_data with the actual result (Data-only JSON stripped by InvokableRun).
			// MCP fallback: when no runtime extra was cached (non-ToolCallResult output), use raw JSON only.
			var extra map[string]any
			if runtimeExtra, ok := rt.toolRuntimeExtras.LoadAndDelete(callID); ok {
				if m, ok := runtimeExtra.(map[string]any); ok {
					extra = m
					if result != "" {
						var rawData json.RawMessage
						if json.Unmarshal([]byte(result), &rawData) == nil {
							extra["raw_data"] = rawData
						}
					}
				}
			} else if result != "" {
				var rawData json.RawMessage
				if json.Unmarshal([]byte(result), &rawData) == nil {
					extra = map[string]any{"raw_data": rawData}
				}
			}

			rt.toolEvtsMu.Lock()
			toolEvt, ok := rt.toolEvts[callID]
			rt.toolEvtsMu.Unlock()
			if ok {
				toolEvt.OnEnd(ctx, result, extra)
			}
			// Close the iteration step when all tools are done.
			if atomic.AddInt32(&rt.pendingToolCalls, -1) == 0 {
				rt.currentStepEvt.OnFinish(ctx)
			}
			return ctx
		},

		// OnError fires when tool execution itself returns an error.
		// Delegates to the cached ToolCallEvent.OnError to close the tool and persist.
		// Fires STEP_FINISHED when this is the last pending tool in the iteration.
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			callID := compose.GetToolCallID(ctx)
			rt.toolEvtsMu.Lock()
			toolEvt, ok := rt.toolEvts[callID]
			rt.toolEvtsMu.Unlock()
			if ok {
				toolEvt.OnError(ctx, err)
			}
			// Close the iteration step when all tools are done.
			if atomic.AddInt32(&rt.pendingToolCalls, -1) == 0 {
				rt.currentStepEvt.OnFinish(ctx)
			}
			return ctx
		},
	}

	modelHandler := &cbtemplate.ModelCallbackHandler{
		// OnEndWithStreamOutput fires inside the ChatModel node goroutine as the LLM
		// produces tokens. Reading here drives real-time AG-UI delivery; Eino fans the
		// stream so the branch checker and outer loop each get their own copy.
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*einomodel.CallbackOutput]) context.Context {
			defer output.Close()
			// Assign a new model-call iteration ID. All events from this iteration
			// (step, reasoning, tool_call, tool_result, assistant_text) share this ID
			// in the DB step_id column.
			rt.currentStepID = uuid.GenerateShortUUID()
			// Create the iteration StepEvent and open the step BEFORE reasoning begins,
			// so the frontend groups the full iteration under one step header.
			rt.currentStepEvt = aievent.NewStepEvent(rt.writer, rt.agent.convSrv, rt.convCtx)
			rt.currentStepEvt.OnStart(ctx, rt.currentStepID, "Reasoning")
			// Reset the pending tool counter; Eino guarantees all tool callbacks from
			// the previous iteration have finished before this callback fires.
			atomic.StoreInt32(&rt.pendingToolCalls, 0)

			reasoning := rt.assistantEvt.NewReasoningEvent(rt.currentStepID)
			hasTool := false
			for {
				out, err := output.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// Ensure the reasoning bubble is closed so the frontend never
					// shows a permanently open reasoning message.
					reasoning.OnEnd(ctx)
					return ctx
				}
				if out == nil || out.Message == nil {
					continue
				}
				// Reasoning/thinking tokens: AG-UI REASONING_MESSAGE_* lifecycle.
				if rc := out.Message.ReasoningContent; rc != "" {
					reasoning.OnContent(rc)
					continue
				}
				// Close the reasoning channel before the first content token arrives.
				reasoning.OnEnd(ctx) // idempotent: no-op if already closed
				// Detect whether this iteration includes tool calls so we know whether
				// to close the step here (text-only) or defer to the tool callbacks.
				if len(out.Message.ToolCalls) > 0 {
					hasTool = true
				}
				if out.Message.Content == "" {
					continue
				}
				rt.assistantEvt.OnContent(out.Message.Content)
			}
			// Flush any trailing reasoning-only model output.
			reasoning.OnEnd(ctx)
			rt.assistantEvt.OnEnd(ctx)
			// Text-only iterations: close the step immediately after model stream ends.
			// Tool iterations: step is closed by the last tool's OnEnd/OnError callback.
			if !hasTool {
				rt.currentStepEvt.OnFinish(ctx)
			}
			return ctx
		},

		// OnError fires on a model-layer failure.
		// Persist any accumulated partial assistant text so that tool-result rows
		// already written by OnEnd have a parent, preventing orphaned tool messages.
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			rt.assistantEvt.Persist(rt.reqCtx, rt.currentStepID)
			return ctx
		},
	}
	return react.BuildAgentCallback(modelHandler, toolHandler)
}

// cloneToolExtra returns a shallow copy of the named tool's ToolInfo.Extra,
// or nil if the tool is not found or has no Extra.
// Used in OnEnd to attach display metadata (icon, render_type) to tool results
// without coupling the callback to specific tool implementations.
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
