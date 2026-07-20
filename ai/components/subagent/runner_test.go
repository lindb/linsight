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
	"sync"
	"testing"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/ai/agui"
	agentapi "github.com/lindb/linsight/ai/api"
	aievent "github.com/lindb/linsight/ai/components/event"
	agentconfig "github.com/lindb/linsight/ai/config"
	agentmodel "github.com/lindb/linsight/ai/model"
)

// scriptedStreamer returns one canned stream per Stream() call, in order.
type scriptedStreamer struct {
	turns [][]*schema.Message
	calls int
}

func (s *scriptedStreamer) Stream(_ context.Context, _ []*schema.Message, _ ...einomodel.Option) (*schema.StreamReader[*schema.Message], error) {
	i := s.calls
	if i >= len(s.turns) {
		i = len(s.turns) - 1
	}
	s.calls++
	return schema.StreamReaderFromArray(s.turns[i]), nil
}

// recordingWriter records the (agentID) of every frame so we can assert tagging.
// It implements agentapi.SSEWriter; Scoped returns a child sharing the same sink.
type recordingWriter struct {
	mu     *sync.Mutex
	sink   *[]string // agentID captured per frame
	agent  string
	parent string
}

func newRecordingWriter() *recordingWriter {
	return &recordingWriter{mu: &sync.Mutex{}, sink: &[]string{}}
}
func (w *recordingWriter) rec() {
	w.mu.Lock()
	defer w.mu.Unlock()
	*w.sink = append(*w.sink, w.agent)
}
func (w *recordingWriter) WriteRunStarted(_, _ string)                             {}
func (w *recordingWriter) WriteRunFinished(_, _, _ string)                         {}
func (w *recordingWriter) WriteRunError(_ string, _ agui.ErrorCode, _ string)      {}
func (w *recordingWriter) WriteRunInterrupt(_, _ string, _ *agui.InterruptPayload) {}
func (w *recordingWriter) WriteTextMessageStart(_ string, _ schema.RoleType)       { w.rec() }
func (w *recordingWriter) WriteTextMessageContent(_, _ string)                     { w.rec() }
func (w *recordingWriter) WriteTextMessageEnd(_ string, _ int64)                   { w.rec() }
func (w *recordingWriter) WriteReasoningMessageStart(_ string)                     { w.rec() }
func (w *recordingWriter) WriteReasoningMessageContent(_, _ string)                { w.rec() }
func (w *recordingWriter) WriteReasoningMessageEnd(_ string, _ int64)              { w.rec() }
func (w *recordingWriter) WriteToolCallStart(_, _, _ string, _ map[string]any)     { w.rec() }
func (w *recordingWriter) WriteToolCallArgs(_, _ string)                           { w.rec() }
func (w *recordingWriter) WriteToolCallEnd(_ string)                               { w.rec() }
func (w *recordingWriter) WriteToolCallResult(_, _, _ string, _ agui.MessageStatus, _ string, _ map[string]any, _ int64) {
	w.rec()
}
func (w *recordingWriter) WriteMessagesSnapshot(_ []agui.Message) {}
func (w *recordingWriter) WriteStepStarted(_, _ string)           { w.rec() }
func (w *recordingWriter) WriteStepFinished(_ string, _ int64)    { w.rec() }
func (w *recordingWriter) Scoped(agentID, parentToolCallID string) agentapi.SSEWriter {
	return &recordingWriter{mu: w.mu, sink: w.sink, agent: agentID, parent: parentToolCallID}
}

// recordingSaver captures persisted events.
type recordingSaver struct {
	mu    sync.Mutex
	saved []*agentmodel.Event
}

func (s *recordingSaver) SaveEvent(_ context.Context, evt *agentmodel.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = append(s.saved, evt)
	return nil
}

func newTestScope(streamer *scriptedStreamer) (*Scope, *recordingWriter, *recordingSaver) {
	w := newRecordingWriter()
	saver := &recordingSaver{}
	return &Scope{
		Writer:    w,
		Saver:     saver,
		ConvCtx:   aievent.NewConvContext(1, "conv-1", 7, "run-1"),
		BaseTools: nil,
	}, w, saver
}

func testCfg() *agentconfig.Config {
	return &agentconfig.Config{MaxStep: 4, MaxParallelToolCalls: 4, MaxSubAgentDepth: 3}
}

// TestRunner_TextOnly verifies a text-only response terminates the loop and returns the text,
// and that all emitted events/frames are tagged with the sub-agent's AgentID.
func TestRunner_TextOnly(t *testing.T) {
	streamer := &scriptedStreamer{turns: [][]*schema.Message{
		{{Role: schema.Assistant, Content: "final answer"}},
	}}
	scope, w, saver := newTestScope(streamer)
	r := NewRunner(streamer, testCfg())

	ctx := WithScope(context.Background(), scope)
	ctx = WithParentToolCallID(ctx, "call-1")

	out, err := r.Run(ctx, RunRequest{AgentID: "analyst", SystemPrompt: "sys", Goal: "do it", Mode: ModeHandoff})
	assert.NoError(t, err)
	assert.Equal(t, "final answer", out)

	// Every SSE frame carries the sub-agent's AgentID.
	assert.NotEmpty(t, *w.sink)
	for _, id := range *w.sink {
		assert.Equal(t, "analyst", id)
	}
	// Every persisted event is tagged with AgentID + ParentToolCallID and shares the run's seq.
	assert.NotEmpty(t, saver.saved)
	for _, e := range saver.saved {
		assert.Equal(t, "analyst", e.AgentID)
		assert.Equal(t, "call-1", e.ParentToolCallID)
		assert.Equal(t, "run-1", e.RunID)
	}
}

// TestRunner_CycleDetected verifies re-entering an already-active agent is refused.
func TestRunner_CycleDetected(t *testing.T) {
	streamer := &scriptedStreamer{turns: [][]*schema.Message{{{Role: schema.Assistant, Content: "x"}}}}
	scope, _, _ := newTestScope(streamer)
	r := NewRunner(streamer, testCfg())

	ctx := WithScope(context.Background(), scope)
	ctx = pushAgent(ctx, "analyst") // analyst already on the stack

	_, err := r.Run(ctx, RunRequest{AgentID: "analyst", SystemPrompt: "sys", Goal: "g", Mode: ModeHandoff})
	assert.ErrorContains(t, err, "cycle detected")
}

// TestRunner_MaxDepth verifies the depth guard refuses runs beyond MaxSubAgentDepth.
func TestRunner_MaxDepth(t *testing.T) {
	streamer := &scriptedStreamer{turns: [][]*schema.Message{{{Role: schema.Assistant, Content: "x"}}}}
	scope, _, _ := newTestScope(streamer)
	cfg := testCfg()
	cfg.MaxSubAgentDepth = 2
	r := NewRunner(streamer, cfg)

	ctx := WithScope(context.Background(), scope)
	ctx = pushAgent(ctx, "a")
	ctx = pushAgent(ctx, "b") // depth 2 already

	_, err := r.Run(ctx, RunRequest{AgentID: "c", SystemPrompt: "sys", Goal: "g", Mode: ModeHandoff})
	assert.ErrorContains(t, err, "max nesting depth")
}

// TestRunner_NoScope verifies a missing scope is a clean error (caller falls back).
func TestRunner_NoScope(t *testing.T) {
	streamer := &scriptedStreamer{turns: [][]*schema.Message{{{Role: schema.Assistant, Content: "x"}}}}
	r := NewRunner(streamer, testCfg())
	_, err := r.Run(context.Background(), RunRequest{AgentID: "a", Mode: ModeHandoff})
	assert.ErrorContains(t, err, "no scope")
}

// echoTool is a minimal InvokableTool used to exercise the tool-execution loop.
type echoTool struct {
	name   string
	called *int32
	mu     *sync.Mutex
}

func (e *echoTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: e.name}, nil
}
func (e *echoTool) InvokableRun(_ context.Context, args string, _ ...tool.Option) (string, error) {
	e.mu.Lock()
	*e.called++
	e.mu.Unlock()
	return "tool-result:" + args, nil
}

// TestRunner_ToolLoop verifies the runner executes a tool call then terminates on the
// following text-only turn, wiring the tool result back into the conversation.
func TestRunner_ToolLoop(t *testing.T) {
	idx := 0
	streamer := &scriptedStreamer{turns: [][]*schema.Message{
		// Turn 1: emit a tool call.
		{{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{
			ID:       "tc-1",
			Index:    &idx,
			Type:     "function",
			Function: schema.FunctionCall{Name: "echo", Arguments: `{"q":1}`},
		}}}},
		// Turn 2: final text.
		{{Role: schema.Assistant, Content: "done"}},
	}}
	scope, _, saver := newTestScope(streamer)

	var called int32
	scope.BaseTools = []tool.BaseTool{&echoTool{name: "echo", called: &called, mu: &sync.Mutex{}}}

	r := NewRunner(streamer, testCfg())
	ctx := WithScope(context.Background(), scope)

	out, err := r.Run(ctx, RunRequest{AgentID: "worker", SystemPrompt: "sys", Goal: "g", Mode: ModeHandoff})
	assert.NoError(t, err)
	assert.Equal(t, "done", out)
	assert.Equal(t, int32(1), called, "tool should be invoked exactly once")

	// A tool_call and tool_result event should have been persisted, tagged with the agent.
	var haveToolCall, haveToolResult bool
	for _, e := range saver.saved {
		switch e.Type {
		case agentmodel.EventTypeToolCall:
			haveToolCall = true
		case agentmodel.EventTypeToolResult:
			haveToolResult = true
		}
		assert.Equal(t, "worker", e.AgentID)
	}
	assert.True(t, haveToolCall, "expected a tool_call event")
	assert.True(t, haveToolResult, "expected a tool_result event")
}
