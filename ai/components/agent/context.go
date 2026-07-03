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

	"github.com/lindb/linsight/ai/agui"
	"github.com/lindb/linsight/ai/components/memory"
	agentmodel "github.com/lindb/linsight/ai/model"
)

// buildLLMMessages assembles the []*schema.Message slice for the LLM from either the
// MemoryStore (if enabled) or a direct DB history load.
func (a *agentRuntime) buildLLMMessages(ctx context.Context, conv *agentmodel.Conversation, _ string) ([]*schema.Message, error) {
	var messages []*schema.Message

	if a.memoryStore != nil && a.cfg.EnableMultiTurnContext {
		agentLog.Info("memory: using MemoryStore for multi-turn context",
			logger.String("conv.uid", conv.UID))

		state, stateErr := a.memoryStore.GetConversationState(ctx, conv.UID)
		if stateErr != nil {
			agentLog.Warn("memory: failed to restore conversation state, falling back to DB history",
				logger.String("conv.uid", conv.UID), logger.Error(stateErr))
			history, histErr := a.convSrv.GetRecentMessages(ctx, conv.ID, a.cfg.MemorySize)
			if histErr != nil {
				return nil, fmt.Errorf("load history fallback: %w", histErr)
			}
			var assembleErr error
			messages, assembleErr = a.assembleMessages(history)
			if assembleErr != nil {
				return nil, assembleErr
			}
		} else {
			agentLog.Info("memory: conversation state restored",
				logger.String("conv.uid", conv.UID),
				logger.Int("contextWindow", len(state.ContextWindow)))
			messages = a.buildMessagesWithMemoryState(state)
		}
	} else {
		agentLog.Info("memory: MemoryStore disabled, using DB history")
		history, histErr := a.convSrv.GetRecentMessages(ctx, conv.ID, a.cfg.MemorySize)
		if histErr != nil {
			return nil, fmt.Errorf("load history: %w", histErr)
		}
		var assembleErr error
		messages, assembleErr = a.assembleMessages(history)
		if assembleErr != nil {
			return nil, assembleErr
		}
	}
	return messages, nil
}

// assembleMessages builds the LLM message slice from DB history rows.
// RawMessage stores eino schema.Message JSON (snake_case keys), so direct
// unmarshal to schema.Message is lossless — ToolCalls, ToolCallID, and ToolName
// are preserved exactly.
func (a *agentRuntime) assembleMessages(history []*agentmodel.Message) ([]*schema.Message, error) {
	msgs := make([]*schema.Message, 0, len(history)+1)

	systemPrompt := a.buildSystemPrompt()
	if systemPrompt != "" {
		msgs = append(msgs, &schema.Message{Role: schema.System, Content: systemPrompt})
	}

	// Collect all tool call_ids referenced by assistant messages so we can detect
	// orphaned tool-result rows (those whose call_id has no matching parent).
	knownCallIDs := make(map[string]bool)
	for _, m := range history {
		if m.Role != schema.Assistant {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			continue
		}
		for _, tc := range sm.ToolCalls {
			if tc.ID != "" {
				knownCallIDs[tc.ID] = true
			}
		}
	}

	for _, m := range history {
		if m.Role == schema.Assistant && m.Status == string(agui.StatusError) {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			agentLog.Warn("skipping malformed history message",
				logger.Int64("id", m.ID), logger.Error(err))
			continue
		}
		// Drop orphaned tool messages; sending them to the LLM causes an API validation
		// error because their ToolCallID references a non-existent parent ToolCall.
		if sm.Role == schema.Tool && !knownCallIDs[sm.ToolCallID] {
			agentLog.Warn("skipping orphaned tool message",
				logger.Int64("id", m.ID), logger.String("tool_call_id", sm.ToolCallID))
			continue
		}
		// Strip reasoning_content: OpenAI-compatible APIs reject messages that carry
		// reasoning tokens in history, returning a 400 validation error.
		sm.ReasoningContent = ""
		msgs = append(msgs, &sm)
	}
	return msgs, nil
}

// buildMessagesWithMemoryState builds messages using MemoryStore conversation state.
// The state's ContextWindow already contains the current user message (saved before
// GetConversationState was called).
func (a *agentRuntime) buildMessagesWithMemoryState(state *memory.ConversationState) []*schema.Message {
	messages := make([]*schema.Message, 0, len(state.ContextWindow)+1)

	systemPrompt := a.buildSystemPrompt()
	if systemPrompt != "" {
		messages = append(messages, &schema.Message{Role: schema.System, Content: systemPrompt})
		preview := systemPrompt
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

// buildSystemPrompt returns the configured base system prompt.
func (a *agentRuntime) buildSystemPrompt() string {
	return a.cfg.SystemPrompt
}
