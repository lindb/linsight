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

	"github.com/gin-gonic/gin"
)

// SSEWriter sends Server-Sent Events to a Gin response writer.
// It is safe for concurrent use: a mutex serialises all writes so that
// multiple goroutines (e.g. the streaming loop and a panic recovery
// handler) do not interleave frames.
type SSEWriter struct {
	mu sync.Mutex
	w  gin.ResponseWriter
}

// NewSSEWriter prepares c for SSE delivery and returns an SSEWriter bound to it.
// It sets the required Content-Type and Cache-Control headers and flushes them
// so the client starts receiving the stream immediately.
func NewSSEWriter(c *gin.Context) *SSEWriter {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Writer.Flush()
	return &SSEWriter{w: c.Writer}
}

// WriteContent sends a content fragment to the client.
func (w *SSEWriter) WriteContent(text string) error {
	return w.writeEvent(&SSEEvent{Type: SSEEventContent, Content: text})
}

// WriteToolCall notifies the client that the agent is about to invoke a tool.
func (w *SSEWriter) WriteToolCall(toolName, callID, arguments string) error {
	return w.writeEvent(&SSEEvent{
		Type:       SSEEventToolCall,
		ToolName:   toolName,
		ToolCallID: callID,
		Arguments:  arguments,
	})
}

// WriteToolResult delivers the result of a completed tool invocation.
func (w *SSEWriter) WriteToolResult(toolName, callID, result string) error {
	return w.writeEvent(&SSEEvent{
		Type:       SSEEventToolResult,
		ToolName:   toolName,
		ToolCallID: callID,
		Result:     result,
	})
}

// WriteDone sends the terminal done event carrying the conversation UID and,
// for new conversations, the auto-generated title (empty string if unavailable).
func (w *SSEWriter) WriteDone(conversationUID, title string) error {
	return w.writeEvent(&SSEEvent{Type: SSEEventDone, ConversationUID: conversationUID, Title: title})
}

// WriteError sends an error event with a machine-readable code and a human-readable message.
func (w *SSEWriter) WriteError(code SSEErrorCode, msg string) error {
	return w.writeEvent(&SSEEvent{Type: SSEEventError, ErrorCode: code, Content: msg})
}

// writeEvent marshals event to JSON and writes it as a single SSE data frame.
// The frame format is: "data: <json>\n\n"
func (w *SSEWriter) writeEvent(event *SSEEvent) error {
	data, err := json.Marshal(event)
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
