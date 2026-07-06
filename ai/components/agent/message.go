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

// Message persistence layer: user and error messages are persisted here;
// all other event types (assistant text, reasoning, tool calls) are handled
// by their respective event lifecycle objects in the ai/event package.

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

// persistUserMessage saves the incoming user request to both the legacy messages table
// and the new events table, then assigns rt.userMsgUID.
func (rt *chatRun) persistUserMessage(ctx context.Context) error {
	rt.userMsgUID = uuid.GenerateShortUUID()
	userMsg := &schema.Message{
		Role:    schema.User,
		Content: rt.req.Content,
	}

	// Save to legacy messages table for backward compatibility with existing
	// multi-turn context assembly that reads from that table.
	raw, _ := json.Marshal(userMsg)
	dbMsg := &agentmodel.Message{
		ConversationID:  rt.conv.ID,
		ConversationUID: rt.conv.UID,
		MessageUID:      rt.userMsgUID,
		Role:            schema.User,
		RawMessage:      json.RawMessage(raw),
		TurnUID:         rt.runID,
		RunUID:          rt.runID,
	}
	dbMsg.CreatedBy = rt.userID
	dbMsg.UpdatedBy = rt.userID
	if err := rt.agent.convSrv.SaveMessage(ctx, dbMsg); err != nil {
		agentLog.Error("failed to save user message", logger.Error(err))
		return fmt.Errorf("save user message: %w", err)
	}

	// Dual-save to the events table so GetConversationEvents can return a
	// complete snapshot without reading from the legacy messages table.
	evt := &agentmodel.Event{
		RunID:      rt.runID,
		ConvUID:    rt.conv.UID,
		MessageUID: rt.userMsgUID,
		Role:       schema.User,
		Type:       agentmodel.EventTypeUserMessage,
		Content:    rt.req.Content,
		RawMessage: json.RawMessage(raw),
		Sequence:   rt.convCtx.NextSeq(),
	}
	evt.CreatedBy = rt.userID
	evt.UpdatedBy = rt.userID
	if err := rt.agent.convSrv.SaveEvent(ctx, evt); err != nil {
		agentLog.Error("failed to save user message event", logger.Error(err))
		return fmt.Errorf("save user message event: %w", err)
	}
	return nil
}

// persistAgentError saves a run_error event so failures are visible after page refresh.
// Uses a fresh background context so this survives request timeout cancellation.
// The event is excluded from LLM context by GetRecentEvents (status=error filter).
func (rt *chatRun) persistAgentError(reqCtx context.Context, chatErr error) {
	code, errMsg := aiutil.ClassifyError(chatErr)

	user := util.GetUser(reqCtx)
	saveCtx, cancel := context.WithTimeout(
		context.WithValue(context.Background(), constant.LinSightSignedKey, user),
		titleGenTimeout,
	)
	defer cancel()

	evt := &agentmodel.Event{
		RunID:      rt.runID,
		ConvUID:    rt.conv.UID,
		MessageUID: uuid.GenerateShortUUID(),
		Role:       schema.Assistant,
		Type:       agentmodel.EventTypeRunError,
		Status:     string(agui.StatusError),
		Content:    errMsg,
		Code:       string(code),
		Sequence:   rt.convCtx.NextSeq(),
	}
	evt.CreatedBy = rt.userID
	evt.UpdatedBy = rt.userID

	if saveErr := rt.agent.convSrv.SaveEvent(saveCtx, evt); saveErr != nil {
		agentLog.Warn("failed to persist run error event", logger.Error(saveErr))
	}
}
