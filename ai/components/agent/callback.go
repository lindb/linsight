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

	"github.com/cloudwego/eino/callbacks"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	cbtemplate "github.com/cloudwego/eino/utils/callbacks"

	"github.com/lindb/linsight/ai/agui"
	"github.com/lindb/linsight/pkg/uuid"
)

// newSSECallback constructs the Eino callbacks.Handler that:
//   - relays model token events as AG-UI SSE frames (TEXT_MESSAGE_*, REASONING_MESSAGE_*)
//   - relays tool invocation events as AG-UI SSE frames (TOOL_CALL_*)
//   - persists per-tool assistant rows and tool-result rows to the DB
func newSSECallback(rt *chatRun) callbacks.Handler {
	toolHandler := &cbtemplate.ToolCallbackHandler{
		// OnStart fires when Eino is about to invoke a tool.
		// Persists the tool-call decision as a role=assistant message so that:
		//   1. GetConversationMessages snapshots tool calls correctly.
		//   2. assembleMessages populates knownCallIDs, preventing all subsequent
		//      tool-result rows from being dropped as orphans.
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
			args := ""
			if input != nil {
				args = input.ArgumentsInJSON
			}
			// compose.GetToolCallID reads the call_id injected by Eino before the
			// callback fires, replacing the previous sync.Map lookup (race-prone).
			callID := compose.GetToolCallID(ctx)

			toolCallMsgUID := uuid.GenerateShortUUID()
			rt.persistToolCall(rt.timeoutCtx, toolCallMsgUID, callID, info.Name, args)

			rt.writer.WriteToolCallStart(callID, info.Name, rt.assistantMsgUID)
			rt.writer.WriteToolCallArgs(callID, args)
			rt.writer.WriteToolCallEnd(callID)
			return ctx
		},

		// OnEnd fires when a tool returns successfully.
		// Emits a TOOL_CALL_RESULT and persists the result row with structured guidance
		// so the frontend can render it with a rich component (table, list, …).
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
			result := ""
			if output != nil {
				result = output.Response
			}
			callID := compose.GetToolCallID(ctx)

			// Build guidance: attempt JSON parse so the frontend can use a rich renderer.
			// Falls back to nil (plain-text display) when result is not valid JSON.
			var guidance *agui.Guidance
			if result != "" {
				var rawData json.RawMessage
				if json.Unmarshal([]byte(result), &rawData) == nil {
					guidance = &agui.Guidance{
						RenderType: inferRenderType(info.Name),
						RawData:    rawData,
					}
				}
			}

			toolMsgUID := uuid.GenerateShortUUID()
			rt.writer.WriteToolCallResult(toolMsgUID, callID, info.Name, agui.StatusComplete, result, guidance)
			rt.persistToolResult(rt.timeoutCtx, toolMsgUID, callID, info.Name, result, guidance)
			return ctx
		},

		// OnError fires when tool execution itself returns an error.
		// Emits a TOOL_CALL_RESULT with the error text so the frontend's tool-call
		// lifecycle is properly closed, and persists the record for multi-turn context.
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			callID := compose.GetToolCallID(ctx)
			errContent := err.Error()
			toolMsgUID := uuid.GenerateShortUUID()
			errGuidance := &agui.Guidance{
				RenderType: "error",
				RawData:    json.RawMessage(fmt.Sprintf(`{"error_msg":%s}`, jsonString(errContent))),
			}
			rt.writer.WriteToolCallResult(toolMsgUID, callID, info.Name, agui.StatusError, errContent, errGuidance)
			rt.persistToolError(rt.timeoutCtx, toolMsgUID, callID, info.Name, errContent, errGuidance)
			return ctx
		},
	}

	modelHandler := &cbtemplate.ModelCallbackHandler{
		// OnEndWithStreamOutput fires inside the ChatModel node goroutine as the LLM
		// produces tokens. Reading here drives real-time AG-UI delivery; Eino fans the
		// stream so the branch checker and outer loop each get their own copy.
		OnEndWithStreamOutput: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[*einomodel.CallbackOutput]) context.Context {
			defer output.Close()
			// reasoningStarted is local: each model invocation has its own independent
			// reasoning phase and must not share state across calls.
			var reasoningStarted bool
			reasoningMsgUID := uuid.GenerateShortUUID()
			for {
				out, err := output.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// Ensure the reasoning bubble is closed so the frontend never
					// shows a permanently open reasoning message.
					if reasoningStarted {
						rt.writer.WriteReasoningMessageEnd(reasoningMsgUID)
					}
					return ctx
				}
				if out == nil || out.Message == nil {
					continue
				}
				// Reasoning/thinking tokens: AG-UI REASONING_MESSAGE_* lifecycle.
				// Reasoning and text messages are independent lifecycles per AG-UI spec.
				if rc := out.Message.ReasoningContent; rc != "" {
					if !reasoningStarted {
						rt.writer.WriteReasoningMessageStart(reasoningMsgUID)
						reasoningStarted = true
					}
					rt.rsb.WriteString(rc)
					rt.writer.WriteReasoningMessageContent(reasoningMsgUID, rc)
					continue
				}
				// Close the reasoning channel before the first content token arrives.
				if reasoningStarted {
					rt.writer.WriteReasoningMessageEnd(reasoningMsgUID)
					reasoningStarted = false
				}
				if out.Message.Content == "" {
					continue
				}
				if !rt.textStarted {
					rt.writer.WriteTextMessageStart(rt.assistantMsgUID, schema.Assistant)
					rt.textStarted = true
				}
				rt.sb.WriteString(out.Message.Content)
				rt.writer.WriteTextMessageContent(rt.assistantMsgUID, out.Message.Content)
			}
			// Flush any trailing reasoning-only model output (model reasoned but produced
			// no content tokens in this call).
			if reasoningStarted {
				rt.writer.WriteReasoningMessageEnd(reasoningMsgUID)
			}
			if rt.textStarted {
				rt.writer.WriteTextMessageEnd(rt.assistantMsgUID)
				// Reset so the error-recovery defer in run() does not double-close.
				rt.textStarted = false
			}
			return ctx
		},

		// OnError fires on a model-layer failure.
		// Save any accumulated partial assistant message so that tool-result rows
		// already written by OnEnd have a parent, preventing orphaned tool messages
		// from breaking the next LLM API call.
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			if (rt.sb.Len() > 0 || rt.rsb.Len() > 0) && !rt.partialSaved {
				rt.partialSaved = true
				rt.persistAssistantContent(rt.reqCtx)
			}
			return ctx
		},
	}
	return react.BuildAgentCallback(modelHandler, toolHandler)
}

// inferRenderType maps a tool name to the suggested frontend render component.
// This drives guidance.render_type so the client picks the right rich renderer
// without inspecting the raw_data structure itself.
func inferRenderType(toolName string) string {
	switch toolName {
	case "linsight_datasource_query":
		return "table"
	case "linsight_list_datasources", "linsight_search_dashboards":
		return "list"
	default:
		return "text"
	}
}

// jsonString encodes s as a JSON string literal.
// Used to safely embed string values inside hand-crafted JSON raw_data objects.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
