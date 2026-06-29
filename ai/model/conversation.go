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

// MessageRole distinguishes who produced a message in the conversation history.
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
)

// Message is a single turn in a conversation, stored for replay as LLM context.
// ToolCalls and ToolCallID mirror the OpenAI function-calling schema so that the
// full assistant turn (including tool invocations) can be reconstructed accurately.
type Message struct {
	basemodel.BaseModel
	ConversationID int64       `json:"conversationId" gorm:"column:conversation_id;index"`
	Role           MessageRole `json:"role" gorm:"column:role"`
	// Content holds the plain-text portion of the message.
	Content    string `json:"content" gorm:"column:content;type:text"`
	// ToolCalls is a JSON-encoded array of tool call requests produced by the assistant.
	// Non-empty only when Role == "assistant" and the model invoked tools.
	ToolCalls  string `json:"toolCalls,omitempty" gorm:"column:tool_calls;type:text"`
	// ToolCallID correlates a tool-result message with the assistant's tool_call request.
	// Non-empty only when Role == "tool".
	ToolCallID string `json:"toolCallId,omitempty" gorm:"column:tool_call_id"`
	// ToolName is the name of the tool that produced this message.
	// Non-empty only when Role == "tool".
	ToolName   string `json:"toolName,omitempty" gorm:"column:tool_name"`
}

// TableName specifies the GORM table name.
func (Message) TableName() string { return "messages" }
