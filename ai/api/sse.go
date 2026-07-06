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
	"sync"

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

	// Streaming text message
	WriteTextMessageStart(messageID string, role schema.RoleType)
	WriteTextMessageContent(messageID, delta string)
	WriteTextMessageEnd(messageID string, elapsedMs int64)

	// Reasoning phase (AG-UI spec: REASONING_MESSAGE_* events)
	WriteReasoningMessageStart(messageID string)
	WriteReasoningMessageContent(messageID, delta string)
	WriteReasoningMessageEnd(messageID string, elapsedMs int64)

	// Tool invocation
	WriteToolCallStart(toolCallID, toolName, parentMessageID string)
	WriteToolCallArgs(toolCallID, argsDelta string)
	WriteToolCallEnd(toolCallID string)
	WriteToolCallResult(messageID, toolCallID, toolName string, status agui.MessageStatus, content string, guidance *agui.Guidance, elapsedMs int64)

	// Conversation history snapshot
	WriteMessagesSnapshot(messages []agui.Message)

	// Step lifecycle: wraps a named execution step (typically one tool invocation).
	// STEP_STARTED fires before TOOL_CALL_START; STEP_FINISHED fires after TOOL_CALL_RESULT.
	WriteStepStarted(stepID, stepName string)
	WriteStepFinished(stepID string, elapsedMs int64)
}

// ginSSEWriter is the concrete SSEWriter implementation backed by gin.ResponseWriter.
type ginSSEWriter struct {
	mu sync.Mutex
	w  gin.ResponseWriter
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
	return &ginSSEWriter{w: c.Writer}
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

func (w *ginSSEWriter) WriteToolCallStart(toolCallID, toolName, parentMessageID string) {
	_ = w.writeEvent(agui.Event{
		Type:            agui.ToolCallStart,
		ToolCallID:      toolCallID,
		ToolCallName:    toolName,
		ParentMessageID: parentMessageID,
		Status:          agui.StatusProcessing,
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

func (w *ginSSEWriter) WriteToolCallResult(messageID, toolCallID, toolName string, status agui.MessageStatus, content string, guidance *agui.Guidance, elapsedMs int64) {
	_ = w.writeEvent(agui.Event{
		Type:         agui.ToolCallResult,
		MessageID:    messageID,
		ToolCallID:   toolCallID,
		ToolCallName: toolName,
		Content:      content,
		Guidance:     guidance,
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
func (w *ginSSEWriter) writeEvent(evt agui.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err = fmt.Fprintf(w.w, "data: %s\n\n", data); err != nil {
		return err
	}
	w.w.Flush()
	return nil
}
