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
	"errors"
	"io"

	"github.com/cloudwego/eino/schema"
)

// fullResponseStreamToolCallChecker detects tool calls in the streaming response.
// It returns early (false) as soon as the model emits reasoning content or regular text
// content, which signals the model is in text-answer mode and will not produce tool calls.
// Returning early prevents Eino from buffering the entire LLM stream before the outer
// reader loop can start, enabling real-time token delivery to the client.
//
// Decision table:
//
//	ToolCalls present          → true  (route to tools node)
//	ReasoningContent non-empty → false (thinking mode; model will not call tools)
//	Content non-empty          → false (text-answer mode; model will not call tools)
//	Empty chunk / prefix       → continue reading
//
// Trade-off: if a model emits text content THEN tool calls in the same turn, the tool
// calls will be missed.  This is acceptable because the OnEndWithStreamOutput callback
// (see buildSSECallback) streams text to the client in real-time for every model call,
// so the two mechanisms complement each other.
//
// The handler closes sr before returning as required by the Eino contract.
func fullResponseStreamToolCallChecker(_ context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
	defer sr.Close()
	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if len(msg.ToolCalls) > 0 {
			return true, nil
		}
		// Once reasoning tokens appear the model is in long-form answer mode — no tool calls follow.
		if msg.ReasoningContent != "" {
			return false, nil
		}
		// Once text content appears the model is producing a text answer, not a tool call.
		if len(msg.Content) > 0 {
			return false, nil
		}
	}
}
