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

package model

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"

	basemodel "github.com/lindb/linsight/model"
)

// Conversation is a named chat session belonging to a user in an org.
type Conversation struct {
	basemodel.BaseModel
	UID    string `json:"uid" gorm:"column:uid;uniqueIndex"`
	OrgID  int64  `json:"orgId" gorm:"column:org_id;index"`
	UserID int64  `json:"userId" gorm:"column:user_id;index"`
	Title  string `json:"title" gorm:"column:title"`
}

// TableName specifies the GORM table name.
func (Conversation) TableName() string { return "conversations" }

// RoleError is a custom message role for persisting agent-side failures as
// conversation messages so they remain visible after a page refresh.
// This role is intentionally NOT in eino's schema package — it is an internal
// storage convention only. Memory and agent code must filter it out before
// sending messages to the LLM.
const RoleError schema.RoleType = "error"

// Message stores a single turn in a conversation.
// RawMessage holds the complete eino schema.Message JSON (snake_case keys), enabling
// the memory store to unmarshal it directly to schema.Message without any field mapping.
// The AG-UI message ID is stored in the separate MessageUID column because schema.Message
// has no "id" field; the service layer reconstructs agui.Message.ID from MessageUID.
type Message struct {
	basemodel.BaseModel
	ConversationID int64 `json:"-" gorm:"column:conversation_id;index"`
	// ConversationUID is the public identifier of the owning conversation.
	// Stored as a real column (not gorm:"-") to allow direct query by UID
	// without an extra join.
	ConversationUID string `json:"conversationUid,omitempty" gorm:"column:conversation_uid;index"`
	// MessageUID is the AG-UI messageId. Stored as a separate column because
	// schema.Message has no "id" field. Used for O(1) deduplication and fast
	// lookup, and to reconstruct agui.Message.ID when serving conversation history.
	MessageUID string `json:"-" gorm:"column:message_uid;uniqueIndex"`
	// Role is a redundant column that mirrors schema.Message.role so queries
	// like WHERE role != 'error' never need to parse JSON.
	Role schema.RoleType `json:"-" gorm:"column:role;index"`
	// RawMessage holds the complete eino schema.Message JSON (snake_case keys).
	// MySQL/PostgreSQL: stored as native JSON column; SQLite: stored as TEXT.
	RawMessage json.RawMessage `json:"rawMessage" gorm:"column:raw_message;type:json"`
	// ToolCallID is a first-class physical column for tool-result messages, mirroring
	// schema.Message.ToolCallID. Stored separately to avoid JSON parsing in queries
	// (e.g. building the guidanceMap in GetAgUiHistory).
	ToolCallID string `json:"-" gorm:"column:tool_call_id;index"`
	// Guidance stores the structured render hint for tool-result messages.
	// Populated by the agent callback when a tool returns queryable structured data.
	// MySQL/PostgreSQL: native JSON column; SQLite: TEXT.
	Guidance json.RawMessage `json:"-" gorm:"column:guidance;type:json"`
	// TurnUID groups all messages belonging to the same logical turn
	// (one user request → tool chain → final assistant reply).
	// Empty for messages written before this column was introduced.
	TurnUID string `json:"-" gorm:"column:turn_uid;index"`
	// RunUID identifies the specific agent execution instance.
	// Currently 1:1 with TurnUID; stored separately for future retry/sub-task support.
	RunUID string `json:"-" gorm:"column:run_uid;index"`
}

// TableName specifies the GORM table name.
func (Message) TableName() string { return "messages" }
