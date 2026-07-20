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

	contextcomp "github.com/lindb/linsight/ai/components/context"
	agentmodel "github.com/lindb/linsight/ai/model"
)

// buildLLMMessages assembles the []*schema.Message slice for the LLM from either the
// ContextProvider (if enabled) or a direct DB history load.
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

	if a.contextProvider != nil && a.cfg.EnableMultiTurnContext {
		agentLog.Info("context: using ContextProvider for multi-turn context",
			logger.String("conv.uid", conv.UID))

		state, stateErr := a.contextProvider.GetConversationState(ctx, conv.UID)
		if stateErr != nil {
			agentLog.Warn("context: failed to restore conversation state, falling back to DB history",
				logger.String("conv.uid", conv.UID), logger.Error(stateErr))
			history, histErr := a.convSrv.GetRecentEvents(ctx, conv.UID, a.cfg.ContextSize)
			if histErr != nil {
				return nil, fmt.Errorf("load history fallback: %w", histErr)
			}
			messages = a.assembleMessages(history)
		} else {
			agentLog.Info("context: conversation state restored",
				logger.String("conv.uid", conv.UID),
				logger.Int("contextWindow", len(state.ContextWindow)))
			messages = a.buildMessagesWithContextState(state)
		}
	} else {
		agentLog.Info("context: ContextProvider disabled, using DB history")
		history, histErr := a.convSrv.GetRecentEvents(ctx, conv.UID, a.cfg.ContextSize)
		if histErr != nil {
			return nil, fmt.Errorf("load history: %w", histErr)
		}
		messages = a.assembleMessages(history)
	}

	// Enforce the tool_call/tool_result pairing invariant exactly once, after the raw message
	// list is assembled and the pending interrupt (if any) is known. This drops unanswered
	// tool_calls / orphaned tool messages and re-appends the pending interrupt's tool_calls.
	messages = contextcomp.SanitizeMessages(conv.UID, messages, pendingToolCalls)

	// Log the exact history context handed to the LLM so tool-call/result pairing problems
	// (which surface as opaque 400 errors from the provider) can be diagnosed from logs alone.
	agentLog.Info("prompt: history context assembled",
		logger.String("conv.uid", conv.UID),
		logger.Int("total_messages", len(messages)),
		logger.Int("pending_tool_calls", len(pendingToolCalls)),
		logger.String("summary", contextcomp.SummarizeMessages(messages)))
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

// historyMessages returns the persisted conversation history as LLM messages, WITHOUT a
// system prompt. Used to seed a fork_with_context sub-agent run that inherits the parent
// conversation. GetRecentEvents already excludes reasoning/step/error rows and (after the
// sub-agent tagging change) sub-agent process events, so only host-level turns are inherited.
func (a *agentRuntime) historyMessages(ctx context.Context, convUID string) ([]*schema.Message, error) {
	history, err := a.convSrv.GetRecentEvents(ctx, convUID, a.cfg.ContextSize)
	if err != nil {
		return nil, err
	}
	msgs := make([]*schema.Message, 0, len(history))
	for _, m := range history {
		if len(m.RawMessage) == 0 {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			continue
		}
		sm.ReasoningContent = ""
		msgs = append(msgs, &sm)
	}
	// Enforce the tool_call/tool_result pairing invariant: drop the in-flight tool_call that
	// spawned this fork (persisted before its result exists) and any other orphans, so the
	// forked sub-agent receives a provider-valid message list.
	msgs = contextcomp.SanitizeMessages(convUID, msgs, nil)
	return msgs, nil
}

// buildMessagesWithContextState builds messages using ContextProvider conversation state.
// The state's ContextWindow already contains the current user message (saved before
// GetConversationState was called).
func (a *agentRuntime) buildMessagesWithContextState(state *contextcomp.ConversationState) []*schema.Message {
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
