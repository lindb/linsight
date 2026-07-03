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

// Message persistence layer: one method per message type, mirroring the SSEWriter
// interface's WriteXxx naming convention so each call site is self-documenting.

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	aiutil "github.com/lindb/linsight/ai/util"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// msgOption is a functional option applied to an agentmodel.Message after construction.
type msgOption func(*agentmodel.Message)

// withMsgStatus sets the status column on the DB message.
// Pass agui.StatusError to mark an agent-side failure that must never be sent to the LLM.
func withMsgStatus(status agui.MessageStatus) msgOption {
	return func(m *agentmodel.Message) {
		m.Status = string(status)
	}
}

// withMsgToolCallID sets the tool_call_id column on the DB message (used for tool-result rows).
func withMsgToolCallID(callID string) msgOption {
	return func(m *agentmodel.Message) {
		m.ToolCallID = callID
	}
}

// withMsgGuidance serializes guidance and attaches it to the DB message.
// A nil guidance is a no-op.
func withMsgGuidance(guidance *agui.Guidance) msgOption {
	return func(m *agentmodel.Message) {
		if guidance != nil {
			m.Guidance, _ = json.Marshal(guidance)
		}
	}
}

// newDBMessage converts a schema.Message into an agentmodel.Message ready for DB persistence.
// The schema.Message is JSON-serialized as-is into RawMessage so round-trips are lossless
// (ToolCalls, ToolCallID, ReasoningContent, etc. are preserved exactly).
// userID is stamped into both CreatedBy and UpdatedBy; opts can override any field afterwards.
func newDBMessage(
	convID int64, convUID string,
	userID int64, msgUID string,
	sm *schema.Message, turnUID string,
	opts ...msgOption,
) *agentmodel.Message {
	raw, _ := json.Marshal(sm)
	m := &agentmodel.Message{
		ConversationID:  convID,
		ConversationUID: convUID,
		MessageUID:      msgUID,
		Role:            sm.Role,
		RawMessage:      json.RawMessage(raw),
		TurnUID:         turnUID,
		RunUID:          turnUID,
	}
	m.CreatedBy = userID
	m.UpdatedBy = userID
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// persistUserMessage saves the incoming user request to the DB and assigns rt.userMsgUID.
func (rt *chatRun) persistUserMessage(ctx context.Context) error {
	rt.userMsgUID = uuid.GenerateShortUUID()
	userMsg := &schema.Message{
		Role:    schema.User,
		Content: rt.req.Content,
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, rt.userMsgUID, userMsg, rt.runID)
	if err := rt.agent.convSrv.SaveMessage(ctx, dbMsg); err != nil {
		agentLog.Error("failed to save user message", logger.Error(err))
		return fmt.Errorf("save user message: %w", err)
	}
	return nil
}

// persistAssistantContent saves the accumulated text and reasoning content as a single
// assistant DB row. Tool calls are intentionally omitted — each was already persisted
// by persistToolCall at the moment the LLM decided to invoke it.
//
// When called from an error path (e.g. OnError callback, run() defers), ctx may already
// be cancelled (client disconnect, request timeout). In that case a fresh background
// context is created so partial content is still persisted for multi-turn history.
func (rt *chatRun) persistAssistantContent(ctx context.Context) {
	saveCtx := ctx
	if ctx.Err() != nil {
		// ctx is already done — create a resilient context so the DB write survives.
		user := util.GetUser(ctx)
		var cancel context.CancelFunc
		saveCtx, cancel = context.WithTimeout(
			context.WithValue(context.Background(), constant.LinSightSignedKey, user),
			titleGenTimeout,
		)
		defer cancel()
	}

	msg := &schema.Message{
		Role:             schema.Assistant,
		Content:          rt.sb.String(),
		ReasoningContent: rt.rsb.String(),
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, rt.assistantMsgUID, msg, rt.runID)
	if err := rt.agent.convSrv.SaveMessage(saveCtx, dbMsg); err != nil {
		agentLog.Warn("failed to persist assistant message", logger.Error(err))
	}
}

// persistToolCall saves a role=assistant tool-call decision row.
// Called before the tool executes so that assembleMessages can populate knownCallIDs,
// preventing subsequent tool-result rows from being dropped as orphans.
func (rt *chatRun) persistToolCall(ctx context.Context, msgUID, callID, toolName, args string) {
	assistantMsg := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID:   callID,
			Type: "function",
			Function: schema.FunctionCall{
				Name:      toolName,
				Arguments: args,
			},
		}},
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, msgUID, assistantMsg, rt.runID)
	if err := rt.agent.convSrv.SaveMessage(ctx, dbMsg); err != nil {
		agentLog.Warn("failed to persist tool-call assistant message",
			logger.String("tool", toolName), logger.Error(err))
	}
}

// persistToolResult saves a successful role=tool result row.
func (rt *chatRun) persistToolResult(ctx context.Context, msgUID, callID, toolName, result string, guidance *agui.Guidance) {
	toolMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    result,
		ToolCallID: callID,
		ToolName:   toolName,
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, msgUID,
		toolMsg, rt.runID,
		withMsgToolCallID(callID),
		withMsgGuidance(guidance))
	if err := rt.agent.convSrv.SaveMessage(ctx, dbMsg); err != nil {
		agentLog.Warn("failed to persist tool result",
			logger.String("tool", toolName), logger.Error(err))
	} else {
		agentLog.Info("stream: tool result persisted",
			logger.String("tool", toolName), logger.Int("result_len", len(result)))
	}
}

// persistToolError saves a failed role=tool result row with status=error.
func (rt *chatRun) persistToolError(ctx context.Context, msgUID, callID, toolName, errContent string, guidance *agui.Guidance) {
	toolMsg := &schema.Message{
		Role:       schema.Tool,
		Content:    errContent,
		ToolCallID: callID,
		ToolName:   toolName,
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, msgUID,
		toolMsg, rt.runID,
		withMsgStatus(agui.StatusError),
		withMsgToolCallID(callID),
		withMsgGuidance(guidance))
	if saveErr := rt.agent.convSrv.SaveMessage(ctx, dbMsg); saveErr != nil {
		agentLog.Warn("failed to persist tool error result",
			logger.String("tool", toolName), logger.Error(saveErr))
	} else {
		agentLog.Info("stream: tool error persisted",
			logger.String("tool", toolName), logger.Int("error_len", len(errContent)))
	}
}

// persistAgentError saves a role=assistant error message so failures are visible after
// page refresh. Uses a fresh background context so this survives request timeout cancellation.
// The message is filtered out of LLM context by assembleMessages (status=error check).
func (rt *chatRun) persistAgentError(reqCtx context.Context, turnUID string, chatErr error) {
	user := util.GetUser(reqCtx)
	saveCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	_, errMsg := aiutil.ClassifyError(chatErr)
	msgUID := uuid.GenerateShortUUID()
	errSchemaMsg := &schema.Message{
		Role:    schema.Assistant,
		Content: errMsg,
	}
	dbMsg := newDBMessage(rt.conv.ID, rt.conv.UID, rt.userID, msgUID, errSchemaMsg, turnUID,
		withMsgStatus(agui.StatusError))
	if saveErr := rt.agent.convSrv.SaveMessage(saveCtx, dbMsg); saveErr != nil {
		agentLog.Warn("failed to persist error message", logger.Error(saveErr))
	}
}
