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
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
)

// ContextProvider builds the LLM context window from the authoritative data source
// (DB event table). It exposes a subset of key-value operations sufficient for
// conversation state retrieval, but the backing store is READ-ONLY: Set / Delete are
// no-ops because the DB event table is the single source of truth for context.
//
// This is NOT an LLM "memory" system — it does NOT persist, compress, or summarise
// knowledge across sessions. It simply derives the context window from persisted
// conversation events and enriches it for the LLM API call.
type ContextProvider interface {
	// Basic key-value operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	ListKeys(ctx context.Context, pattern string) ([]string, error)

	// Conversation state management extensions
	GetConversationState(ctx context.Context, conversationID string) (*ConversationState, error)
	SetConversationState(ctx context.Context, conversationID string, state *ConversationState) error
}

// ConversationState is a snapshot of the conversation context used to construct the
// LLM context window for multi-turn dialogue.
type ConversationState struct {
	ConversationID string            `json:"conversationId"`
	OrgID          int64             `json:"orgId"`
	UserID         int64             `json:"userId"`
	LastActiveAt   time.Time         `json:"lastActiveAt"`
	ContextWindow  []*schema.Message `json:"contextWindow"`
	ActiveToolCalls []*ToolCall      `json:"activeToolCalls"`
	Metadata       map[string]string `json:"metadata"`
}

// ToolCall is a tool invocation record.
type ToolCall struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Arguments string    `json:"arguments"`
	Result    string    `json:"result"`
	CalledAt  time.Time `json:"calledAt"`
	Status    string    `json:"status"` // pending, completed, failed
}

// ProviderType is the context provider backend type.
type ProviderType string

const (
	ProviderTypeDB    ProviderType = "db"
	ProviderTypeRedis ProviderType = "redis"
)

// DefaultProviderType is the default context provider type.
const DefaultProviderType = ProviderTypeDB

// DefaultTTL is the default time-to-live.
const DefaultTTL = 24 * time.Hour

// Key patterns for conversation state management.
const (
	KeyPatternConversationState   = "conv:%s:state"
	KeyPatternConversationContext = "conv:%s:context"
	KeyPatternActiveConversation  = "org:%d:conv:%s:active"
)

// BuildConversationStateKey builds the conversation state key.
func BuildConversationStateKey(conversationID string) string {
	return fmt.Sprintf(KeyPatternConversationState, conversationID)
}

// BuildConversationContextKey builds the conversation context key.
func BuildConversationContextKey(conversationID string) string {
	return fmt.Sprintf(KeyPatternConversationContext, conversationID)
}

// BuildActiveConversationKey builds the active conversation key.
func BuildActiveConversationKey(orgID int64, conversationID string) string {
	return fmt.Sprintf(KeyPatternActiveConversation, orgID, conversationID)
}
