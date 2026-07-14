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

package context

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"

	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

var dbContextLog = logger.GetLogger("agent", "db_context_provider")

// dbContextProvider implements ContextProvider backed by the database event table.
// It derives the LLM context window from persisted conversation events — it does NOT
// maintain its own mutable state. Set / Delete are no-ops because the event table is
// the single source of truth.
type dbContextProvider struct {
	db          dbpkg.DB
	convSrv     ConversationProvider
	contextSize int
}

// NewDBContextProvider creates a database-backed ContextProvider instance.
func NewDBContextProvider(db dbpkg.DB, convSrv ConversationProvider, contextSize int) ContextProvider {
	return &dbContextProvider{
		db:          db,
		convSrv:     convSrv,
		contextSize: contextSize,
	}
}

// Get retrieves the value for the specified key.
func (s *dbContextProvider) Get(ctx context.Context, key string) ([]byte, error) {
	// For conversation state keys, retrieve from ConversationProvider
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		return s.getConversationState(ctx, key)
	}

	// For other keys, use key-value storage table (future extension)
	return nil, fmt.Errorf("key type not supported: %s", key)
}

// Set is a no-op: the DB event table is the authoritative source of context.
// Context is derived on read, never written through this interface.
func (s *dbContextProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		dbContextLog.Debug("context: state write ignored (read-only, derived from DB events)",
			logger.String("key", key))
		return nil
	}
	return fmt.Errorf("key type not supported: %s", key)
}

// Delete is a no-op: context is derived from DB events and cannot be deleted
// through this interface.
func (s *dbContextProvider) Delete(ctx context.Context, key string) error {
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		dbContextLog.Debug("context: state delete ignored (read-only, derived from DB events)",
			logger.String("key", key))
		return nil
	}
	return fmt.Errorf("key type not supported: %s", key)
}

// Exists checks if the key exists.
func (s *dbContextProvider) Exists(ctx context.Context, key string) (bool, error) {
	if strings.HasPrefix(key, "conv:") && strings.HasSuffix(key, ":state") {
		_, err := s.getConversationState(ctx, key)
		return err == nil, nil
	}

	return false, fmt.Errorf("key type not supported: %s", key)
}

// ListKeys lists all keys matching the pattern.
func (s *dbContextProvider) ListKeys(ctx context.Context, pattern string) ([]string, error) {
	// Currently only supports conversation state key listing
	if pattern == "conv:*:state" {
		return s.listConversationStateKeys(ctx)
	}

	return nil, fmt.Errorf("pattern not supported: %s", pattern)
}

// GetConversationState retrieves conversation state from the event table.
func (s *dbContextProvider) GetConversationState(ctx context.Context, conversationID string) (*ConversationState, error) {
	key := BuildConversationStateKey(conversationID)
	data, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var state ConversationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal conversation state: %w", err)
	}

	dbContextLog.Info("context: state loaded from DB events",
		logger.String("conv.uid", conversationID),
		logger.Int("contextWindow", len(state.ContextWindow)),
	)
	return &state, nil
}

// SetConversationState is a no-op: state is derived from DB events on read.
func (s *dbContextProvider) SetConversationState(ctx context.Context, conversationID string, state *ConversationState) error {
	key := BuildConversationStateKey(conversationID)
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal conversation state: %w", err)
	}

	return s.Set(ctx, key, data, DefaultTTL)
}

// getConversationState retrieves conversation state from ConversationProvider.
func (s *dbContextProvider) getConversationState(ctx context.Context, key string) ([]byte, error) {
	conversationID := extractConversationIDFromKey(key)
	if conversationID == "" {
		return nil, fmt.Errorf("invalid conversation state key: %s", key)
	}

	conv, err := s.convSrv.GetConversation(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	messages, err := s.convSrv.GetRecentEvents(ctx, conv.UID, s.contextSize)
	if err != nil {
		return nil, fmt.Errorf("get recent events: %w", err)
	}

	dbContextLog.Info("context: building state from DB events",
		logger.String("conv.uid", conversationID),
		logger.Int64("conv.id", conv.ID),
		logger.Int("events", len(messages)),
	)

	state := &ConversationState{
		ConversationID: conversationID,
		OrgID:          conv.OrgID,
		UserID:         conv.UserID,
		LastActiveAt:   time.Now(),
		ContextWindow:  s.convertEventsToSchema(messages),
		Metadata:       make(map[string]string),
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("marshal state: %w", err)
	}

	return data, nil
}

// listConversationStateKeys lists all conversation state keys.
func (s *dbContextProvider) listConversationStateKeys(ctx context.Context) ([]string, error) {
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

// convertEventsToSchema converts DB event rows to schema.Message for LLM context.
// GetRecentEvents already excludes reasoning, step, and error-status rows, so this
// method only needs to handle user/assistant/tool events.
//
// The tool_call/tool_result pairing invariant is NOT enforced here; SanitizeMessages is
// applied once at the single assembly entry point (agentRuntime.buildLLMMessages), where
// the pending-interrupt tool calls are also known. Enforcing it here as well would strip
// an in-flight interrupt's tool_calls before the resume path can re-append them.
func (s *dbContextProvider) convertEventsToSchema(events []*agentmodel.Event) []*schema.Message {
	result := make([]*schema.Message, 0, len(events))
	for _, evt := range events {
		if len(evt.RawMessage) == 0 {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(evt.RawMessage, &sm); err != nil {
			dbContextLog.Warn("context: skipping event with malformed raw_message",
				logger.Int64("evt.id", evt.ID),
				logger.Error(err),
			)
			continue
		}
		result = append(result, &sm)
	}
	return result
}

// extractConversationIDFromKey extracts conversation ID from key name.
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
