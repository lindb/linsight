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
	"fmt"
	"time"

	"github.com/cloudwego/eino/schema"
)

// MemoryStore Eino-standard key-value storage interface for conversation state management
type MemoryStore interface {
	// Basic key-value operations
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	ListKeys(ctx context.Context, pattern string) ([]string, error)

	// Conversation state management extensions
	GetConversationState(ctx context.Context, conversationID string) (*ConversationState, error)
	SetConversationState(ctx context.Context, conversationID string, state *ConversationState) error
	LockConversation(ctx context.Context, conversationID string) error
	UnlockConversation(ctx context.Context, conversationID string) error
}

// ConversationState Conversation state structure for persisting multi-turn dialogue context
type ConversationState struct {
	ConversationID   string            `json:"conversationId"`
	OrgID           int64             `json:"orgId"`
	UserID          int64             `json:"userId"`
	LastActiveAt    time.Time         `json:"lastActiveAt"`
	ContextWindow   []*schema.Message `json:"contextWindow"`
	ActiveToolCalls []*ToolCall       `json:"activeToolCalls"`
	Metadata        map[string]string `json:"metadata"`
}

// ToolCall Tool invocation record
type ToolCall struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Arguments string    `json:"arguments"`
	Result    string    `json:"result"`
	CalledAt  time.Time `json:"calledAt"`
	Status    string    `json:"status"` // pending, completed, failed
}

// StoreType Storage backend type
type StoreType string

const (
	StoreTypeDB    StoreType = "db"
	StoreTypeRedis StoreType = "redis"
)

// DefaultStoreType Default storage type
const DefaultStoreType = StoreTypeDB

// DefaultTTL Default time-to-live
const DefaultTTL = 24 * time.Hour

// Key patterns for conversation state management
const (
	KeyPatternConversationState = "conv:%s:state"
	KeyPatternConversationLock  = "conv:%s:lock"
	KeyPatternConversationContext = "conv:%s:context"
	KeyPatternActiveConversation = "org:%d:conv:%s:active"
)

// BuildConversationStateKey Builds conversation state key
func BuildConversationStateKey(conversationID string) string {
	return fmt.Sprintf(KeyPatternConversationState, conversationID)
}

// BuildConversationLockKey Builds conversation lock key
func BuildConversationLockKey(conversationID string) string {
	return fmt.Sprintf(KeyPatternConversationLock, conversationID)
}

// BuildConversationContextKey Builds conversation context key
func BuildConversationContextKey(conversationID string) string {
	return fmt.Sprintf(KeyPatternConversationContext, conversationID)
}

// BuildActiveConversationKey Builds active conversation key
func BuildActiveConversationKey(orgID int64, conversationID string) string {
	return fmt.Sprintf(KeyPatternActiveConversation, orgID, conversationID)
}