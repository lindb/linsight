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

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentapi "github.com/lindb/linsight/ai/api"
	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/uuid"
)

var toolLog = logger.GetLogger("ai", "ToolCallEvent")

// ToolCallEvent manages the lifecycle of one tool invocation:
//
//	SSE: TOOL_CALL_START → TOOL_CALL_ARGS → TOOL_CALL_END → TOOL_CALL_RESULT
//	DB:  tool_call row → tool_result row
//
// The STEP_STARTED/STEP_FINISHED lifecycle is managed by StepEvent (one per
// model-call iteration), which wraps the full iteration including this tool call.
// modelStepID is the model-call iteration ID from chatRun.currentStepID; stored
// in the DB step_id column to correlate all events from the same ReAct iteration.
type ToolCallEvent struct {
	callID       string
	toolName     string
	parentMsgUID string
	modelStepID  string // model-call iteration ID; stored in DB step_id column
	startTime    time.Time

	writer  agentapi.SSEWriter
	saver   EventSaver
	convCtx *ConvContext
}

// NewToolCallEvent allocates a ToolCallEvent for one tool invocation.
// callID is the tool_call_id assigned by the LLM.
// parentMsgUID links the tool call to its parent assistant message.
// modelStepID should be chatRun.currentStepID (the model-call iteration ID).
func NewToolCallEvent(
	callID, toolName, parentMsgUID, modelStepID string,
	writer agentapi.SSEWriter,
	saver EventSaver,
	convCtx *ConvContext,
) *ToolCallEvent {
	return &ToolCallEvent{
		callID:       callID,
		toolName:     toolName,
		parentMsgUID: parentMsgUID,
		modelStepID:  modelStepID,
		writer:       writer,
		saver:        saver,
		convCtx:      convCtx,
	}
}

// OnStart fires when Eino is about to invoke the tool.
// Emits TOOL_CALL_START + TOOL_CALL_ARGS + TOOL_CALL_END and persists a tool_call
// DB row so downstream tool_result rows are never orphaned.
// STEP_STARTED is NOT emitted here — it is managed by StepEvent for the full iteration.
func (e *ToolCallEvent) OnStart(ctx context.Context, args string) {
	e.startTime = time.Now()

	// SSE: tool call lifecycle.
	e.writer.WriteToolCallStart(e.callID, e.toolName, e.parentMsgUID)
	e.writer.WriteToolCallArgs(e.callID, args)
	e.writer.WriteToolCallEnd(e.callID)

	// DB: tool_call row — preserves the LLM's decision for context reconstruction.
	assistantMsg := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:   e.callID,
			Type: "function",
			Function: schema.FunctionCall{
				Name:      e.toolName,
				Arguments: args,
			},
		}},
	}
	toolCallEvt := newDBEvent(e.convCtx,
		agentmodel.EventTypeToolCall,
		e.modelStepID,
		schema.Assistant,
		"", // no plain-text content for tool_call rows
		0,
		withToolCallID(e.callID),
		withToolName(e.toolName),
		withRawMessage(assistantMsg),
	)
	if err := e.saver.SaveEvent(ctx, toolCallEvt); err != nil {
		toolLog.Warn("failed to persist tool_call event",
			logger.String("tool", e.toolName), logger.Error(err))
	}
}

// OnEnd fires when the tool returns successfully.
// Emits TOOL_CALL_RESULT + STEP_FINISHED and persists tool_result + step_finished DB rows.
func (e *ToolCallEvent) OnEnd(ctx context.Context, result string, guidance *agui.Guidance) {
	elapsed := time.Since(e.startTime).Milliseconds()

	toolMsgUID := uuid.GenerateShortUUID()
	e.writer.WriteToolCallResult(toolMsgUID, e.callID, e.toolName, agui.StatusComplete, result, guidance)

	// DB: tool_result row.
	toolMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    result,
		ToolCallID: e.callID,
		ToolName:   e.toolName,
	}
	toolResultEvt := newDBEvent(e.convCtx,
		agentmodel.EventTypeToolResult,
		e.modelStepID,
		schema.Tool,
		result,
		elapsed,
		withToolCallID(e.callID),
		withToolName(e.toolName),
		withGuidance(guidance),
		withRawMessage(toolMsg),
	)
	if err := e.saver.SaveEvent(ctx, toolResultEvt); err != nil {
		toolLog.Warn("failed to persist tool_result event",
			logger.String("tool", e.toolName), logger.Error(err))
	} else {
		toolLog.Info("stream: tool result persisted",
			logger.String("tool", e.toolName), logger.Int("result_len", len(result)))
	}
}

// OnError fires when tool execution returns an error.
// Emits TOOL_CALL_RESULT (status=error) + STEP_FINISHED and persists error rows.
func (e *ToolCallEvent) OnError(ctx context.Context, err error) {
	elapsed := time.Since(e.startTime).Milliseconds()
	errContent := err.Error()

	toolMsgUID := uuid.GenerateShortUUID()
	errGuidance := &agui.Guidance{
		RenderType: "error",
		RawData:    json.RawMessage(fmt.Sprintf(`{"error_msg":%s}`, jsonString(errContent))),
	}
	e.writer.WriteToolCallResult(toolMsgUID, e.callID, e.toolName, agui.StatusError, errContent, errGuidance)

	// DB: tool_result row with error status.
	toolMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    errContent,
		ToolCallID: e.callID,
		ToolName:   e.toolName,
	}
	toolResultEvt := newDBEvent(e.convCtx,
		agentmodel.EventTypeToolResult,
		e.modelStepID,
		schema.Tool,
		errContent,
		elapsed,
		withStatus(agui.StatusError),
		withToolCallID(e.callID),
		withToolName(e.toolName),
		withGuidance(errGuidance),
		withRawMessage(toolMsg),
	)
	if err2 := e.saver.SaveEvent(ctx, toolResultEvt); err2 != nil {
		toolLog.Warn("failed to persist tool_result error event",
			logger.String("tool", e.toolName), logger.Error(err2))
	} else {
		toolLog.Info("stream: tool error persisted",
			logger.String("tool", e.toolName), logger.Int("error_len", len(errContent)))
	}
}

// jsonString encodes s as a JSON string literal.
// Used to safely embed string values inside hand-crafted JSON raw_data objects.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
