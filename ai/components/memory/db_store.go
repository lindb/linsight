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

package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

var dbMemoryLog = logger.GetLogger("agent", "db_memory_store")

// DBMemoryStore MemoryStore implementation based on database, adapting existing ConversationProvider
type dbMemoryStore struct {
	db         dbpkg.DB
	convSrv    ConversationProvider
	memorySize int
}

// NewDBMemoryStore creates a database-based MemoryStore instance
func NewDBMemoryStore(db dbpkg.DB, convSrv ConversationProvider, memorySize int) MemoryStore {
	return &dbMemoryStore{
		db:         db,
		convSrv:    convSrv,
		memorySize: memorySize,
	}
}

// Get retrieves the value for the specified key
func (s *dbMemoryStore) Get(ctx context.Context, key string) ([]byte, error) {
	// For conversation state keys, retrieve from ConversationProvider
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		return s.getConversationState(ctx, key)
	}

	// For other keys, use key-value storage table (future extension)
	return nil, fmt.Errorf("key type not supported: %s", key)
}

// Set sets the key-value pair
func (s *dbMemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// For conversation state keys, update Conversation state
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		return s.setConversationState(ctx, key, value)
	}

	return fmt.Errorf("key type not supported: %s", key)
}

// Delete deletes the specified key
func (s *dbMemoryStore) Delete(ctx context.Context, key string) error {
	// For conversation state keys, clear Conversation state
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		return s.deleteConversationState(ctx, key)
	}

	return fmt.Errorf("key type not supported: %s", key)
}

// Exists checks if the key exists
func (s *dbMemoryStore) Exists(ctx context.Context, key string) (bool, error) {
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		_, err := s.getConversationState(ctx, key)
		return err == nil, nil
	}

	return false, fmt.Errorf("key type not supported: %s", key)
}

// ListKeys lists all keys matching the pattern
func (s *dbMemoryStore) ListKeys(ctx context.Context, pattern string) ([]string, error) {
	// Currently only supports conversation state key listing
	if pattern == "conv:*:state" {
		return s.listConversationStateKeys(ctx)
	}

	return nil, fmt.Errorf("pattern not supported: %s", pattern)
}

// GetConversationState retrieves conversation state
func (s *dbMemoryStore) GetConversationState(ctx context.Context, conversationID string) (*ConversationState, error) {
	key := BuildConversationStateKey(conversationID)
	data, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var state ConversationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal conversation state: %w", err)
	}

	dbMemoryLog.Info("memory: state loaded from DB",
		logger.String("conv.uid", conversationID),
		logger.Int("contextWindow", len(state.ContextWindow)),
	)
	return &state, nil
}

// SetConversationState sets conversation state
func (s *dbMemoryStore) SetConversationState(ctx context.Context, conversationID string, state *ConversationState) error {
	key := BuildConversationStateKey(conversationID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal conversation state: %w", err)
	}

	return s.Set(ctx, key, data, DefaultTTL)
}

// LockConversation is a no-op for the DB-backed store.
// SQLite does not support SELECT ... FOR UPDATE; its write-lock provides
// sufficient serialization for single-node deployments.
func (s *dbMemoryStore) LockConversation(ctx context.Context, conversationID string) error {
	return nil
}

// UnlockConversation unlocks conversation
func (s *dbMemoryStore) UnlockConversation(ctx context.Context, conversationID string) error {
	dbMemoryLog.Info("memory: conversation unlocked", logger.String("conv.uid", conversationID))
	return nil
}

// getConversationState retrieves conversation state from ConversationProvider
func (s *dbMemoryStore) getConversationState(ctx context.Context, key string) ([]byte, error) {
	conversationID := extractConversationIDFromKey(key)
	if conversationID == "" {
		return nil, fmt.Errorf("invalid conversation state key: %s", key)
	}

	conv, err := s.convSrv.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	messages, err := s.convSrv.GetRecentMessages(ctx, conv.ID, s.memorySize)
	if err != nil {
		return nil, fmt.Errorf("get recent messages: %w", err)
	}

	dbMemoryLog.Info("memory: building state from DB messages",
		logger.String("conv.uid", conversationID),
		logger.Int64("conv.id", conv.ID),
		logger.Int("messages", len(messages)),
	)

	state := &ConversationState{
		ConversationID: conversationID,
		OrgID:         conv.OrgID,
		UserID:        conv.UserID,
		LastActiveAt:  time.Now(),
		ContextWindow: s.convertMessagesToSchema(messages),
		Metadata:      make(map[string]string),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal state: %w", err)
	}

	return data, nil
}

// setConversationState updates conversation state
func (s *dbMemoryStore) setConversationState(ctx context.Context, key string, value []byte) error {
	// Conversation state is read-only, automatically built from ConversationProvider
	// Here we only log state updates, actual state is managed by ConversationProvider
	dbMemoryLog.Debug("conversation state updated", logger.String("key", key))
	return nil
}

// deleteConversationState deletes conversation state
func (s *dbMemoryStore) deleteConversationState(ctx context.Context, key string) error {
	// Conversation state is read-only, delete operation only clears cache
	dbMemoryLog.Debug("conversation state deleted", logger.String("key", key))
	return nil
}

// listConversationStateKeys lists all conversation state keys
func (s *dbMemoryStore) listConversationStateKeys(ctx context.Context) ([]string, error) {
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return nil, fmt.Errorf("user context required")
	}

	// Get all conversations for this organization
	conversations, _, err := s.convSrv.ListConversations(ctx, 0, 1000) // Max 1000 conversations
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	keys := make([]string, 0, len(conversations))
	for _, conv := range conversations {
		keys = append(keys, BuildConversationStateKey(conv.UID))
	}

	return keys, nil
}

// convertMessagesToSchema converts database messages to schema.Message for LLM context.
// RawMessage now stores eino schema.Message JSON natively (snake_case keys), so the
// unmarshal is lossless — ToolCalls, ToolCallID, and ToolName are all preserved.
//
// Verbal-promise assistant messages (no ToolCalls, content < 100 runes) are filtered out
// because they were produced before tool-call persistence was fixed and replaying them
// trains the LLM to respond with text instead of invoking tools.
//
// Orphaned tool messages (ToolCallID not referenced by any assistant ToolCall) are also
// filtered to prevent LLM API validation errors on the next conversation turn.
func (s *dbMemoryStore) convertMessagesToSchema(messages []*agentmodel.Message) []*schema.Message {
	// Collect all call_ids referenced by assistant messages to detect orphaned tool rows.
	knownCallIDs := make(map[string]bool)
	for _, msg := range messages {
		if msg.Role != schema.Assistant {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(msg.RawMessage, &sm); err != nil {
			continue
		}
		for _, tc := range sm.ToolCalls {
			if tc.ID != "" {
				knownCallIDs[tc.ID] = true
			}
		}
	}

	result := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		// Agent-level error messages (role=assistant, status=error) are for display only;
		// never send them to the LLM. Tool-error messages (role=tool, status=error) must
		// remain — the LLM requires every tool_call to be followed by a tool result.
		if msg.Role == schema.Assistant && msg.Status == string(agui.StatusError) {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(msg.RawMessage, &sm); err != nil {
			dbMemoryLog.Warn("memory: skipping message with malformed raw_message",
				logger.Int64("msg.id", msg.ID),
				logger.Error(err),
			)
			continue
		}
		// Drop orphaned tool messages; sending them to the LLM causes an API validation
		// error because their ToolCallID references a non-existent parent ToolCall.
		if sm.Role == schema.Tool && !knownCallIDs[sm.ToolCallID] {
			dbMemoryLog.Warn("memory: skipping orphaned tool message",
				logger.Int64("msg.id", msg.ID),
				logger.String("tool_call_id", sm.ToolCallID),
			)
			continue
		}
		result = append(result, &sm)
	}
	return result
}

// extractConversationIDFromKey extracts conversation ID from key name
func extractConversationIDFromKey(key string) string {
	if !strings.HasPrefix(key, "conv:") || !strings.HasSuffix(key, ":state") {
		return ""
	}

	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		return ""
	}

	return parts[1]
}
