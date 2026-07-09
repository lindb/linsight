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
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/components/memory"
	agentmodel "github.com/lindb/linsight/ai/model"
)

// buildLLMMessages assembles the []*schema.Message slice for the LLM from either the
// MemoryStore (if enabled) or a direct DB history load.
//
// pendingToolCalls carries the tool calls of an unresolved interrupt on a resume run; it is
// nil for ordinary turns. It is forwarded to SanitizeMessages, which re-appends them as the
// trailing assistant message so the resume graph's Case-B guard always matches. Sanitization
// is applied exactly once here (not inside assembleMessages / convertEventsToSchema) so the
// pending calls are known at the single point where the invariant is enforced.
func (a *agentRuntime) buildLLMMessages(ctx context.Context, conv *agentmodel.Conversation,
	pendingToolCalls []schema.ToolCall,
) ([]*schema.Message, error) {
	var messages []*schema.Message

	if a.memoryStore != nil && a.cfg.EnableMultiTurnContext {
		agentLog.Info("memory: using MemoryStore for multi-turn context",
			logger.String("conv.uid", conv.UID))

		state, stateErr := a.memoryStore.GetConversationState(ctx, conv.UID)
		if stateErr != nil {
			agentLog.Warn("memory: failed to restore conversation state, falling back to DB history",
				logger.String("conv.uid", conv.UID), logger.Error(stateErr))
			history, histErr := a.convSrv.GetRecentEvents(ctx, conv.UID, a.cfg.MemorySize)
			if histErr != nil {
				return nil, fmt.Errorf("load history fallback: %w", histErr)
			}
			messages = a.assembleMessages(history)
		} else {
			agentLog.Info("memory: conversation state restored",
				logger.String("conv.uid", conv.UID),
				logger.Int("contextWindow", len(state.ContextWindow)))
			messages = a.buildMessagesWithMemoryState(state)
		}
	} else {
		agentLog.Info("memory: MemoryStore disabled, using DB history")
		history, histErr := a.convSrv.GetRecentEvents(ctx, conv.UID, a.cfg.MemorySize)
		if histErr != nil {
			return nil, fmt.Errorf("load history: %w", histErr)
		}
		messages = a.assembleMessages(history)
	}

	// Enforce the tool_call/tool_result pairing invariant exactly once, after the raw message
	// list is assembled and the pending interrupt (if any) is known. This drops unanswered
	// tool_calls / orphaned tool messages and re-appends the pending interrupt's tool_calls.
	messages = memory.SanitizeMessages(conv.UID, messages, pendingToolCalls)

	// Log the exact history context handed to the LLM so tool-call/result pairing problems
	// (which surface as opaque 400 errors from the provider) can be diagnosed from logs alone.
	agentLog.Info("prompt: history context assembled",
		logger.String("conv.uid", conv.UID),
		logger.Int("total_messages", len(messages)),
		logger.Int("pending_tool_calls", len(pendingToolCalls)),
		logger.String("summary", memory.SummarizeMessages(messages)))
	return messages, nil
}

// assembleMessages builds the LLM message slice from DB event rows.
// RawMessage stores eino schema.Message JSON (snake_case keys), so direct
// unmarshal to schema.Message is lossless — ToolCalls, ToolCallID, and ToolName
// are preserved exactly.
// GetRecentEvents already excludes reasoning, step, and error-status rows at query level.
// The tool_call/tool_result pairing invariant is enforced once by the caller
// (buildLLMMessages via SanitizeMessages), not here.
func (a *agentRuntime) assembleMessages(history []*agentmodel.Event) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(history)+1)

	sysPrompt := a.buildSystemPrompt()
	if sysPrompt != "" {
		msgs = append(msgs, &schema.Message{Role: schema.System, Content: sysPrompt})
	}

	for _, m := range history {
		if len(m.RawMessage) == 0 {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			agentLog.Warn("skipping malformed history event",
				logger.Int64("id", m.ID), logger.Error(err))
			continue
		}
		// Strip reasoning_content: OpenAI-compatible APIs reject messages that carry
		// reasoning tokens in history, returning a 400 validation error.
		sm.ReasoningContent = ""
		msgs = append(msgs, &sm)
	}
	return msgs
}

// buildMessagesWithMemoryState builds messages using MemoryStore conversation state.
// The state's ContextWindow already contains the current user message (saved before
// GetConversationState was called).
func (a *agentRuntime) buildMessagesWithMemoryState(state *memory.ConversationState) []*schema.Message {
	messages := make([]*schema.Message, 0, len(state.ContextWindow)+1)

	sysPrompt := a.buildSystemPrompt()
	if sysPrompt != "" {
		messages = append(messages, &schema.Message{Role: schema.System, Content: sysPrompt})
		preview := sysPrompt
		if len(preview) > 80 {
			preview = preview[:80] + "..."
		}
		agentLog.Info("prompt: system prompt injected",
			logger.String("conv.uid", state.ConversationID),
			logger.String("preview", preview))
	}

	for _, msg := range state.ContextWindow {
		// Strip reasoning_content: OpenAI-compatible APIs reject messages that carry
		// reasoning tokens in history, returning a 400 validation error.
		msg.ReasoningContent = ""
		messages = append(messages, msg)
	}

	agentLog.Info("prompt: final message list built",
		logger.String("conv.uid", state.ConversationID),
		logger.Int("total_messages", len(messages)),
		logger.Int("history_messages", len(state.ContextWindow)))

	return messages
}

// buildSystemPrompt returns the system prompt injected at the head of every
// conversation. It is a structured fault-diagnosis prompt (see system_prompt.go).
func (a *agentRuntime) buildSystemPrompt() string {
	return systemPrompt
}
