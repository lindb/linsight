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

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/ai/agui"
)

// SSEWriter sends AG-UI protocol events to a Gin response writer.
// All methods are safe for concurrent use; a mutex serialises writes so that
// the streaming loop and error-recovery goroutines do not interleave frames.
type SSEWriter interface {
	// Run lifecycle
	WriteRunStarted(runID, conversationUID string)
	WriteRunFinished(runID, conversationUID, title string)
	WriteRunError(runID string, code agui.ErrorCode, message string)
	// WriteRunInterrupt signals that the agent paused for human approval.
	// It sends a RUN_FINISHED frame with Outcome=interrupt and the interrupt payload
	// so the frontend can render the approval card(s) and know which interruptId to echo back.
	WriteRunInterrupt(runID, conversationUID string, payload *agui.InterruptPayload)

	// Streaming text message
	WriteTextMessageStart(messageID string, role schema.RoleType)
	WriteTextMessageContent(messageID, delta string)
	WriteTextMessageEnd(messageID string, elapsedMs int64)

	// Reasoning phase (AG-UI spec: REASONING_MESSAGE_* events)
	WriteReasoningMessageStart(messageID string)
	WriteReasoningMessageContent(messageID, delta string)
	WriteReasoningMessageEnd(messageID string, elapsedMs int64)

	// Tool invocation
	WriteToolCallStart(toolCallID, toolName, parentMessageID string, extra map[string]any)
	WriteToolCallArgs(toolCallID, argsDelta string)
	WriteToolCallEnd(toolCallID string)
	WriteToolCallResult(messageID, toolCallID, toolName string, status agui.MessageStatus, content string, extra map[string]any, elapsedMs int64)

	// Conversation history snapshot
	WriteMessagesSnapshot(messages []agui.Message)

	// Step lifecycle: wraps a named execution step (typically one tool invocation).
	// STEP_STARTED fires before TOOL_CALL_START; STEP_FINISHED fires after TOOL_CALL_RESULT.
	WriteStepStarted(stepID, stepName string)
	WriteStepFinished(stepID string, elapsedMs int64)

	// Scoped returns a writer whose emitted frames all carry the given agentID and
	// parentToolCallID, so a sub-agent run's events can be nested under the host
	// tool_call that spawned it. The returned writer shares the underlying connection
	// and mutex, so frames from the host and all sub-agents remain serialized.
	Scoped(agentID, parentToolCallID string) SSEWriter
}

// ginSSEWriter is the concrete SSEWriter implementation backed by gin.ResponseWriter.
// rc extends the HTTP write deadline on every flush so the connection stays alive
// across long-running agent runs whose wall-clock duration exceeds the server's
// WriteTimeout (default 30s).
type ginSSEWriter struct {
	mu *sync.Mutex
	w  gin.ResponseWriter
	rc *http.ResponseController
	// agentID / parentToolCallID stamp every frame emitted by a Scoped() sub-writer so
	// sub-agent events are attributed and nestable. Empty on the host-level writer.
	agentID          string
	parentToolCallID string
}

// NewSSEWriter prepares c for SSE delivery and returns an SSEWriter bound to it.
// It sets the required Content-Type and Cache-Control headers and flushes them
// so the client starts receiving the stream immediately.
func NewSSEWriter(c *gin.Context) SSEWriter {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.Flush()
	return &ginSSEWriter{
		mu: &sync.Mutex{},
		w:  c.Writer,
		rc: http.NewResponseController(c.Writer),
	}
}

// Scoped returns a shallow copy that stamps agentID/parentToolCallID onto every emitted
// frame. The mutex and connection are shared (by pointer) so host and sub-agent frames
// never interleave mid-frame and the write deadline logic stays coherent.
func (w *ginSSEWriter) Scoped(agentID, parentToolCallID string) SSEWriter {
	return &ginSSEWriter{
		mu:               w.mu,
		w:                w.w,
		rc:               w.rc,
		agentID:          agentID,
		parentToolCallID: parentToolCallID,
	}
}

func (w *ginSSEWriter) WriteRunStarted(runID, conversationUID string) {
	_ = w.writeEvent(agui.Event{
		Type:            agui.RunStarted,
		RunID:           runID,
		ConversationUID: conversationUID,
	})
}

func (w *ginSSEWriter) WriteRunFinished(runID, conversationUID, title string) {
	_ = w.writeEvent(agui.Event{
		Type:            agui.RunFinished,
		RunID:           runID,
		ConversationUID: conversationUID,
		Title:           title,
	})
}

func (w *ginSSEWriter) WriteRunError(runID string, code agui.ErrorCode, message string) {
	_ = w.writeEvent(agui.Event{
		Type:    agui.RunError,
		RunID:   runID,
		Code:    string(code),
		Message: message,
	})
}

func (w *ginSSEWriter) WriteRunInterrupt(runID, conversationUID string, payload *agui.InterruptPayload) {
	_ = w.writeEvent(agui.Event{
		Type:            agui.RunFinished,
		RunID:           runID,
		ConversationUID: conversationUID,
		Outcome:         agui.RunOutcomeInterrupt,
		Interrupt:       payload,
	})
}

func (w *ginSSEWriter) WriteTextMessageStart(messageID string, role schema.RoleType) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.TextMessageStart,
		MessageID: messageID,
		Role:      role,
		Status:    agui.StatusStreaming,
	})
}

func (w *ginSSEWriter) WriteTextMessageContent(messageID, delta string) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.TextMessageContent,
		MessageID: messageID,
		Delta:     delta,
	})
}

func (w *ginSSEWriter) WriteReasoningMessageStart(messageID string) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.ReasoningMessageStart,
		MessageID: messageID,
		Status:    agui.StatusStreaming,
	})
}

func (w *ginSSEWriter) WriteReasoningMessageContent(messageID, delta string) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.ReasoningMessageContent,
		MessageID: messageID,
		Delta:     delta,
	})
}

func (w *ginSSEWriter) WriteReasoningMessageEnd(messageID string, elapsedMs int64) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.ReasoningMessageEnd,
		MessageID: messageID,
		Status:    agui.StatusComplete,
		ElapsedMs: elapsedMs,
	})
}

func (w *ginSSEWriter) WriteTextMessageEnd(messageID string, elapsedMs int64) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.TextMessageEnd,
		MessageID: messageID,
		Status:    agui.StatusComplete,
		ElapsedMs: elapsedMs,
	})
}

func (w *ginSSEWriter) WriteToolCallStart(toolCallID, toolName, parentMessageID string, extra map[string]any) {
	_ = w.writeEvent(agui.Event{
		Type:            agui.ToolCallStart,
		ToolCallID:      toolCallID,
		ToolCallName:    toolName,
		ParentMessageID: parentMessageID,
		Status:          agui.StatusProcessing,
		Extra:           extra,
	})
}

func (w *ginSSEWriter) WriteToolCallArgs(toolCallID, argsDelta string) {
	_ = w.writeEvent(agui.Event{
		Type:       agui.ToolCallArgs,
		ToolCallID: toolCallID,
		Delta:      argsDelta,
	})
}

func (w *ginSSEWriter) WriteToolCallEnd(toolCallID string) {
	_ = w.writeEvent(agui.Event{
		Type:       agui.ToolCallEnd,
		ToolCallID: toolCallID,
	})
}

func (w *ginSSEWriter) WriteToolCallResult(messageID, toolCallID, toolName string, status agui.MessageStatus, content string, extra map[string]any, elapsedMs int64) {
	_ = w.writeEvent(agui.Event{
		Type:         agui.ToolCallResult,
		MessageID:    messageID,
		ToolCallID:   toolCallID,
		ToolCallName: toolName,
		Content:      content,
		Extra:        extra,
		Status:       status,
		ElapsedMs:    elapsedMs,
	})
}

func (w *ginSSEWriter) WriteMessagesSnapshot(messages []agui.Message) {
	_ = w.writeEvent(agui.Event{
		Type:     agui.MessagesSnapshot,
		Messages: messages,
	})
}

func (w *ginSSEWriter) WriteStepStarted(stepID, stepName string) {
	_ = w.writeEvent(agui.Event{
		Type:     agui.StepStarted,
		StepID:   stepID,
		StepName: stepName,
	})
}

func (w *ginSSEWriter) WriteStepFinished(stepID string, elapsedMs int64) {
	_ = w.writeEvent(agui.Event{
		Type:      agui.StepFinished,
		StepID:    stepID,
		ElapsedMs: elapsedMs,
	})
}

// writeEvent marshals evt to JSON and writes it as a single SSE data frame.
// The frame format is: "data: <json>\n\n"
// Before each flush the write deadline is extended by 30s so the connection
// survives long-running agent runs that exceed the server's WriteTimeout.
func (w *ginSSEWriter) writeEvent(evt agui.Event) error {
	// Stamp sub-agent attribution onto every frame emitted by a Scoped() writer.
	// No-op on the host-level writer where both fields are empty.
	if w.agentID != "" {
		evt.AgentID = w.agentID
	}
	if w.parentToolCallID != "" {
		evt.ParentToolCallID = w.parentToolCallID
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Push the write deadline forward so the server does not kill the connection
	// while the agent is still producing events. Default WriteTimeout is 30s;
	// extending by 30s on each flush keeps the connection alive indefinitely
	// as long as events continue to flow.
	if w.rc != nil {
		_ = w.rc.SetWriteDeadline(time.Now().Add(30 * time.Second))
	}

	if _, err = fmt.Fprintf(w.w, "data: %s\n\n", data); err != nil {
		return err
	}
	w.w.Flush()
	return nil
}
