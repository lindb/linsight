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

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/schema"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

var convTestCtx = context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
	Org:  &model.Org{BaseModel: model.BaseModel{ID: 1}},
	User: &model.User{BaseModel: model.BaseModel{ID: 1}},
})

// toolCallRaw builds the RawMessage JSON a tool_call row stores (single-element ToolCalls).
func toolCallRaw(callID, name string) json.RawMessage {
	raw, _ := json.Marshal(&schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{{
			ID: callID, Type: "function",
			Function: schema.FunctionCall{Name: name, Arguments: "{}"},
		}},
	})
	return raw
}

func toolResultRaw(callID, name, content string) json.RawMessage {
	raw, _ := json.Marshal(&schema.Message{
		Role: schema.Tool, Content: content, ToolCallID: callID, ToolName: name,
	})
	return raw
}

func assistantTextRaw(content string) json.RawMessage {
	raw, _ := json.Marshal(&schema.Message{Role: schema.Assistant, Content: content})
	return raw
}

// TestGetConversationEvents_ResumeInterruptOrdering verifies the core snapshot-replay fix:
// after an approved resume reuses the interrupted run's RunID/StepID, the interrupt card must
// be replayed AFTER the interrupt step's tool_call and BEFORE its (post-approval) tool_result,
// and the continuation text must carry the same RunID as the original turn.
func TestGetConversationEvents_ResumeInterruptOrdering(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockDB(ctrl)
	srv := NewConversationService(mockDB)

	const runID = "R1"
	const stepID = "S1"
	const contStepID = "S2"

	// DB rows for one interrupted-then-resumed turn, in global insert order (id asc).
	rows := []*agentmodel.Event{
		{RunID: runID, StepID: stepID, Type: agentmodel.EventTypeStepStarted, ToolName: "Deep Analysis", Sequence: 1},
		{RunID: runID, StepID: stepID, MessageUID: "tc-msg", Type: agentmodel.EventTypeToolCall,
			ToolCallID: "call_1", RawMessage: toolCallRaw("call_1", "query"), Sequence: 2},
		{RunID: runID, StepID: stepID, MessageUID: "asst-1", Type: agentmodel.EventTypeAssistantText,
			Content: "let me check", RawMessage: assistantTextRaw("let me check"), Sequence: 3},
		{RunID: runID, StepID: stepID, Type: agentmodel.EventTypeStepFinished, Sequence: 4},
		// === approve; resume reuses runID/stepID and writes the tool_result ===
		{RunID: runID, StepID: stepID, MessageUID: "tr-msg", Type: agentmodel.EventTypeToolResult,
			ToolCallID: "call_1", Content: "result body", RawMessage: toolResultRaw("call_1", "query", "result body"), Sequence: 5},
		// continuation loops back to a new step under the SAME run
		{RunID: runID, StepID: contStepID, Type: agentmodel.EventTypeStepStarted, ToolName: "Deep Analysis", Sequence: 6},
		{RunID: runID, StepID: contStepID, MessageUID: "asst-2", Type: agentmodel.EventTypeAssistantText,
			Content: "final answer", RawMessage: assistantTextRaw("final answer"), Sequence: 7},
		{RunID: runID, StepID: contStepID, Type: agentmodel.EventTypeStepFinished, Sequence: 8},
	}

	payload := agui.InterruptPayload{
		InterruptID: "int-1",
		ToolCalls:   []agui.PendingToolCall{{ID: "call_1", Name: "query", Arguments: "{}"}},
	}
	payloadJSON, _ := json.Marshal(&payload)
	interruptRecs := []*agentmodel.InterruptRecord{
		{UID: "int-1", ConvUID: "C1", OrigRunID: runID, StepID: stepID,
			Status: agentmodel.InterruptStatusApproved, ClientPayload: payloadJSON},
	}

	mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _ string, _ ...any) error {
			*(out.(*agentmodel.Conversation)) = agentmodel.Conversation{UID: "C1"}
			return nil
		})
	mockDB.EXPECT().FindForPaging(gomock.Any(), 0, gomock.Any(), "id asc", gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _, _ int, _, _ string, _ ...any) error {
			*(out.(*[]*agentmodel.Event)) = rows
			return nil
		})
	mockDB.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _ string, _ ...any) error {
			*(out.(*[]*agentmodel.InterruptRecord)) = interruptRecs
			return nil
		})

	events, err := srv.GetConversationEvents(convTestCtx, "C1", 100)
	assert.NoError(t, err)

	// Locate the key events by type.
	idxInterrupt, idxToolCallStart, idxToolResult, idxContText := -1, -1, -1, -1
	for i, e := range events {
		switch {
		case e.Type == agui.RunFinished && e.Outcome == agui.RunOutcomeInterrupt:
			idxInterrupt = i
		case e.Type == agui.ToolCallStart && e.ToolCallID == "call_1":
			idxToolCallStart = i
		case e.Type == agui.ToolCallResult && e.ToolCallID == "call_1":
			idxToolResult = i
		case e.Type == agui.TextMessageStart && e.MessageID == "asst-2":
			idxContText = i
		}
	}

	assert.NotEqual(t, -1, idxInterrupt, "interrupt frame must be present")
	assert.NotEqual(t, -1, idxToolCallStart, "tool_call start must be present")
	assert.NotEqual(t, -1, idxToolResult, "tool_result must be present")
	assert.NotEqual(t, -1, idxContText, "continuation text must be present")

	// Core ordering: tool_call → interrupt card → tool_result.
	assert.Less(t, idxToolCallStart, idxInterrupt, "interrupt card comes after the tool_call")
	assert.Less(t, idxInterrupt, idxToolResult, "interrupt card comes before the (post-approval) tool_result")

	// Continuation text belongs to the SAME run so the frontend keeps it in the same turn.
	assert.Equal(t, runID, events[idxContText].RunID)

	// The interrupted tool_call's parent is the interrupt step's assistant text, not the
	// continuation step's — verifies step-level parent association.
	assert.Equal(t, "asst-1", events[idxToolCallStart].ParentMessageID)
}

// TestGetConversationEvents_PendingInterruptTail verifies a still-pending interrupt (no
// tool_result rows yet) is appended by the tail fallback so the approval card is not dropped.
func TestGetConversationEvents_PendingInterruptTail(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockDB(ctrl)
	srv := NewConversationService(mockDB)

	rows := []*agentmodel.Event{
		{RunID: "R1", StepID: "S1", MessageUID: "tc-msg", Type: agentmodel.EventTypeToolCall,
			ToolCallID: "call_1", RawMessage: toolCallRaw("call_1", "query"), Sequence: 1},
	}
	payload := agui.InterruptPayload{InterruptID: "int-1", ToolCalls: []agui.PendingToolCall{{ID: "call_1", Name: "query"}}}
	payloadJSON, _ := json.Marshal(&payload)
	interruptRecs := []*agentmodel.InterruptRecord{
		{UID: "int-1", ConvUID: "C1", OrigRunID: "R1", StepID: "S1",
			Status: agentmodel.InterruptStatusPending, ClientPayload: payloadJSON},
	}

	mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _ string, _ ...any) error {
			*(out.(*agentmodel.Conversation)) = agentmodel.Conversation{UID: "C1"}
			return nil
		})
	mockDB.EXPECT().FindForPaging(gomock.Any(), 0, gomock.Any(), "id asc", gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _, _ int, _, _ string, _ ...any) error {
			*(out.(*[]*agentmodel.Event)) = rows
			return nil
		})
	mockDB.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(out any, _ string, _ ...any) error {
			*(out.(*[]*agentmodel.InterruptRecord)) = interruptRecs
			return nil
		})

	events, err := srv.GetConversationEvents(convTestCtx, "C1", 100)
	assert.NoError(t, err)

	last := events[len(events)-1]
	assert.Equal(t, agui.RunFinished, last.Type)
	assert.Equal(t, agui.RunOutcomeInterrupt, last.Outcome)
}

func TestGetMaxEventSequence(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockDB(ctrl)
	srv := NewConversationService(mockDB)

	t.Run("returns max sequence of the run", func(t *testing.T) {
		mockDB.EXPECT().FindForPaging(gomock.Any(), 0, 1, "sequence desc", gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(out any, _, _ int, _, _ string, _ ...any) error {
				*(out.(*[]*agentmodel.Event)) = []*agentmodel.Event{{Sequence: 7}}
				return nil
			})
		n, err := srv.GetMaxEventSequence(convTestCtx, "C1", "R1")
		assert.NoError(t, err)
		assert.Equal(t, 7, n)
	})

	t.Run("returns 0 when the run has no events", func(t *testing.T) {
		mockDB.EXPECT().FindForPaging(gomock.Any(), 0, 1, "sequence desc", gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(out any, _, _ int, _, _ string, _ ...any) error {
				*(out.(*[]*agentmodel.Event)) = nil
				return nil
			})
		n, err := srv.GetMaxEventSequence(convTestCtx, "C1", "R1")
		assert.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}
