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
	// like WHERE role = 'assistant' never need to parse JSON.
	Role schema.RoleType `json:"-" gorm:"column:role;index"`
	// Status is a first-class column for error-state detection without JSON parsing.
	// "error" means the message is an agent-side failure and must never be sent to the LLM.
	Status string `json:"-" gorm:"column:status"`
	// RawMessage holds the complete eino schema.Message JSON (snake_case keys).
	// MySQL/PostgreSQL: stored as native JSON column; SQLite: stored as TEXT.
	RawMessage json.RawMessage `json:"rawMessage" gorm:"column:raw_message;type:json"`
	// ToolCallID is a first-class physical column for tool-result messages, mirroring
	// schema.Message.ToolCallID. Stored separately to avoid JSON parsing in queries
	// (e.g. building the guidanceMap in GetAgUiHistory).
	ToolCallID string `json:"-" gorm:"column:tool_call_id;index"`
	// Extra stores the structured render hints and metadata for tool-result messages.
	// Populated by the agent callback when a tool returns queryable structured data.
	// MySQL/PostgreSQL: native JSON column; SQLite: TEXT.
	Extra json.RawMessage `json:"-" gorm:"column:extra;type:json"`
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

// Event stores one completed AG-UI event with full content, timing, and ordering metadata.
// Text and reasoning are stored as separate rows; tool-call and tool-result are separate rows.
// Step events (step_started, step_finished) wrap tool invocations for execution path replay.
//
// step_id semantics: all events produced within a single model-call iteration share the same
// step_id (generated at the start of OnEndWithStreamOutput). Querying by step_id returns the
// full execution path for one ReAct iteration: reasoning → step_started → tool_call →
// tool_result → step_finished → assistant_text.
// user_message and run_error events are not part of any model-call iteration; step_id is empty.
type Event struct {
	basemodel.BaseModel
	RunID      string          `gorm:"column:run_id;index"`
	ConvUID    string          `gorm:"column:conv_uid;index"`
	MessageUID string          `gorm:"column:message_uid;uniqueIndex"`
	// StepID groups all events produced in one model-call iteration (reasoning → tool → text).
	// Empty for user_message and run_error events.
	StepID     string          `gorm:"column:step_id;index"`
	Role       schema.RoleType `gorm:"column:role;index"`
	// Type is the event category; see EventType* constants.
	Type       string          `gorm:"column:type;index"`
	// Status mirrors agui.MessageStatus; "error" rows are excluded from LLM context.
	Status     string          `gorm:"column:status;index"`
	// Content holds the full text for text/reasoning events, tool result for tool_result,
	// tool name for step_started, and human-readable error message for run_error.
	Content    string          `gorm:"column:content;type:text"`
	// Code holds the machine-readable error code for run_error events (maps to agui.ErrorCode).
	Code       string          `gorm:"column:code"`
	ToolCallID string          `gorm:"column:tool_call_id;index"`
	ToolName   string          `gorm:"column:tool_name"`
	// RawMessage is the schema.Message JSON for LLM context reconstruction (user/assistant/tool roles).
	// Populated for event types that contribute to LLM context; empty for step and reasoning events.
	RawMessage json.RawMessage `gorm:"column:raw_message;type:json"`
	// Extra carries the structured render hints and metadata for tool_result and
	// tool_call events. tool_result: render hints + raw_data from tool execution.
	// tool_call: static ToolInfo.Extra (icon/render_type) for frontend pre-rendering.
	// DB column name kept as "guidance" to avoid schema migration.
	Extra   json.RawMessage `gorm:"column:extra;type:json"`
	// Sequence is the run-scoped insertion order for deterministic replay.
	Sequence   int             `gorm:"column:sequence;default:0"`
	// ElapsedMs is the time from first content to persist (text/reasoning) or from
	// OnStart to OnEnd (tool events). Zero for step_started and user_message rows.
	ElapsedMs  int64           `gorm:"column:elapsed_ms;default:0"`
}

// TableName specifies the GORM table name.
func (Event) TableName() string { return "events" }

// Event type categories stored in the Event.Type column.
const (
	// EventTypeUserMessage is the incoming user request.
	EventTypeUserMessage = "user_message"
	// EventTypeAssistantText is the accumulated text response from one chatRun.
	EventTypeAssistantText = "assistant_text"
	// EventTypeReasoning is a thinking/reasoning block from a single model call.
	// Excluded from LLM context (models reject their own reasoning content).
	EventTypeReasoning = "reasoning"
	// EventTypeToolCall is the LLM's tool-invocation decision (role=assistant, has ToolCalls).
	EventTypeToolCall = "tool_call"
	// EventTypeToolResult is the tool execution result (role=tool).
	EventTypeToolResult = "tool_result"
	// EventTypeStepStarted opens a named execution step (one tool invocation).
	// Content holds the tool name; ElapsedMs is zero.
	EventTypeStepStarted = "step_started"
	// EventTypeStepFinished closes a named execution step.
	// ElapsedMs holds the total tool execution duration in milliseconds.
	EventTypeStepFinished = "step_finished"
	// EventTypeRunError is a terminal error event for the agent run.
	// Code holds agui.ErrorCode; Content holds the human-readable message.
	EventTypeRunError = "run_error"
)
