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
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
)

// textFrame records a single TEXT_MESSAGE_* SSE frame for assertion.
type textFrame struct {
	kind      string // "start" | "content" | "end"
	messageID string
	delta     string
	elapsedMs int64
}

// fakeSSEWriter captures only the TEXT_MESSAGE_* frames (and STEP_* frame counts); every
// other SSEWriter method is a no-op so the fake satisfies the full interface without pulling
// in a gin response writer.
type fakeSSEWriter struct {
	text         []textFrame
	stepStarted  int
	stepFinished int
}

func (w *fakeSSEWriter) WriteTextMessageStart(messageID string, _ schema.RoleType) {
	w.text = append(w.text, textFrame{kind: "start", messageID: messageID})
}
func (w *fakeSSEWriter) WriteTextMessageContent(messageID, delta string) {
	w.text = append(w.text, textFrame{kind: "content", messageID: messageID, delta: delta})
}
func (w *fakeSSEWriter) WriteTextMessageEnd(messageID string, elapsedMs int64) {
	w.text = append(w.text, textFrame{kind: "end", messageID: messageID, elapsedMs: elapsedMs})
}

func (w *fakeSSEWriter) WriteRunStarted(_, _ string)                             {}
func (w *fakeSSEWriter) WriteRunFinished(_, _, _ string)                         {}
func (w *fakeSSEWriter) WriteRunError(_ string, _ agui.ErrorCode, _ string)      {}
func (w *fakeSSEWriter) WriteRunInterrupt(_, _ string, _ *agui.InterruptPayload) {}
func (w *fakeSSEWriter) WriteReasoningMessageStart(_ string)                     {}
func (w *fakeSSEWriter) WriteReasoningMessageContent(_, _ string)                {}
func (w *fakeSSEWriter) WriteReasoningMessageEnd(_ string, _ int64)              {}
func (w *fakeSSEWriter) WriteToolCallStart(_, _, _ string, _ map[string]any)     {}
func (w *fakeSSEWriter) WriteToolCallArgs(_, _ string)                           {}
func (w *fakeSSEWriter) WriteToolCallEnd(_ string)                               {}
func (w *fakeSSEWriter) WriteToolCallResult(_, _, _ string, _ agui.MessageStatus, _ string, _ map[string]any, _ int64) {
}
func (w *fakeSSEWriter) WriteMessagesSnapshot(_ []agui.Message) {}
func (w *fakeSSEWriter) WriteStepStarted(_, _ string)           { w.stepStarted++ }
func (w *fakeSSEWriter) WriteStepFinished(_, _ string, _ int64) { w.stepFinished++ }
func (w *fakeSSEWriter) Scoped(_, _ string) agentapi.SSEWriter  { return w }

// fakeSaver captures persisted DB events.
type fakeSaver struct {
	saved []*agentmodel.Event
}

func (s *fakeSaver) SaveEvent(_ context.Context, evt *agentmodel.Event) error {
	s.saved = append(s.saved, evt)
	return nil
}

func newTestAssistantEvent() (*AssistantTextEvent, *fakeSSEWriter, *fakeSaver) {
	w := &fakeSSEWriter{}
	s := &fakeSaver{}
	e := NewAssistantTextEvent(w, s, NewConvContext(1, "conv-1", 7, "run-1"))
	return e, w, s
}

// TestAssistantTextEvent_SegmentPersistsWithSameID verifies one step's segment emits
// START/CONTENT/END under the supplied msgUID and persists a single assistant_text row
// carrying that SAME msgUID and content — so live SSE and history replay share the ID.
func TestAssistantTextEvent_SegmentPersistsWithSameID(t *testing.T) {
	e, w, s := newTestAssistantEvent()

	e.BeginSegment("msg-A", "step-A")
	assert.Equal(t, "msg-A", e.CurrentMsgUID())
	e.OnContent("Hello ")
	e.OnContent("world")
	e.OnEnd(context.Background())

	// SSE frames: start, content, content, end — all under msg-A.
	assert.Equal(t, []textFrame{
		{kind: "start", messageID: "msg-A"},
		{kind: "content", messageID: "msg-A", delta: "Hello "},
		{kind: "content", messageID: "msg-A", delta: "world"},
		{kind: "end", messageID: "msg-A", elapsedMs: w.text[3].elapsedMs},
	}, w.text)

	// Exactly one persisted row, ID == SSE msgUID, content accumulated, tied to the step.
	assert.Len(t, s.saved, 1)
	row := s.saved[0]
	assert.Equal(t, "msg-A", row.MessageUID)
	assert.Equal(t, "step-A", row.StepID)
	assert.Equal(t, agentmodel.EventTypeAssistantText, row.Type)
	assert.Equal(t, "Hello world", row.Content)
	assert.Equal(t, schema.Assistant, row.Role)
	assert.GreaterOrEqual(t, row.ElapsedMs, int64(0))

	// RawMessage carries the assistant message for LLM context reconstruction.
	assert.NotEmpty(t, row.RawMessage)
}

// TestAssistantTextEvent_PerStepElapsedNotCumulative verifies each step's elapsedMs is
// measured from that step's own first token — not accumulated from the run start (the
// bug that made a late step's TEXT_MESSAGE_END exceed its STEP_FINISHED).
func TestAssistantTextEvent_PerStepElapsedNotCumulative(t *testing.T) {
	e, _, s := newTestAssistantEvent()

	// Step 1: hold the segment open ~30ms before closing.
	e.BeginSegment("msg-1", "step-1")
	e.OnContent("slow step")
	time.Sleep(30 * time.Millisecond)
	e.OnEnd(context.Background())

	// Step 2: close almost immediately.
	e.BeginSegment("msg-2", "step-2")
	e.OnContent("fast step")
	e.OnEnd(context.Background())

	assert.Len(t, s.saved, 2)
	step1, step2 := s.saved[0], s.saved[1]
	assert.Equal(t, "msg-1", step1.MessageUID)
	assert.Equal(t, "msg-2", step2.MessageUID)
	// If elapsed were cumulative, step2 would be >= step1. Per-step timing makes the fast
	// second segment strictly shorter than the slow first one.
	assert.Less(t, step2.ElapsedMs, step1.ElapsedMs,
		"step2 elapsed (%d) should be less than step1 (%d) — elapsed must be per-step, not cumulative",
		step2.ElapsedMs, step1.ElapsedMs)
}

// TestAssistantTextEvent_EmptySegment verifies a step that produced no text emits no
// frames and persists no row (so a tool-only step has no phantom assistant_text row).
func TestAssistantTextEvent_EmptySegment(t *testing.T) {
	e, w, s := newTestAssistantEvent()

	e.BeginSegment("msg-empty", "step-empty")
	e.OnEnd(context.Background())

	assert.Empty(t, w.text)
	assert.Empty(t, s.saved)
	assert.False(t, e.IsStarted())
}

// TestAssistantTextEvent_TwoSegmentsDistinctRows verifies consecutive steps produce two
// independent rows with distinct IDs and their own content — matching the live SSE where
// each step re-opens a text message.
func TestAssistantTextEvent_TwoSegmentsDistinctRows(t *testing.T) {
	e, w, s := newTestAssistantEvent()

	e.BeginSegment("msg-1", "step-1")
	e.OnContent("first")
	e.OnEnd(context.Background())

	e.BeginSegment("msg-2", "step-2")
	e.OnContent("second")
	e.OnEnd(context.Background())

	assert.Len(t, s.saved, 2)
	assert.Equal(t, "first", s.saved[0].Content)
	assert.Equal(t, "second", s.saved[1].Content)
	assert.Equal(t, "msg-1", s.saved[0].MessageUID)
	assert.Equal(t, "msg-2", s.saved[1].MessageUID)

	// Two full START/END pairs across the two segments.
	var starts, ends int
	for _, f := range w.text {
		switch f.kind {
		case "start":
			starts++
		case "end":
			ends++
		}
	}
	assert.Equal(t, 2, starts)
	assert.Equal(t, 2, ends)
}
