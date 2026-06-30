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

package api

import "errors"

// ErrTooManyRequests is returned when the per-org concurrency limit is exceeded.
// Defined here (rather than in the service layer) so that ai/util can reference it
// without creating a circular import (ai/util ← ai/api; ai/service → ai/util).
var ErrTooManyRequests = errors.New("too many concurrent requests for this org")

// ChatRequest is the request body for POST /agent/chat and
// POST /agent/conversations/:uid/chat.
type ChatRequest struct {
	// Content is the user's plain-text message.
	Content string `json:"content" binding:"required"`
}

// SSEEventType identifies the kind of Server-Sent Event frame.
type SSEEventType string

const (
	// SSEEventContent carries a streamed text fragment from the LLM.
	SSEEventContent SSEEventType = "content"
	// SSEEventToolCall notifies the client that the agent is about to invoke a tool.
	SSEEventToolCall SSEEventType = "tool_call"
	// SSEEventToolResult delivers the result of a completed tool invocation.
	SSEEventToolResult SSEEventType = "tool_result"
	// SSEEventDone signals that the full response has been delivered.
	SSEEventDone SSEEventType = "done"
	// SSEEventError signals an unrecoverable error during response generation.
	SSEEventError SSEEventType = "error"
)

// SSEErrorCode categorises the error so clients can show actionable guidance
// rather than a generic error message.
type SSEErrorCode string

const (
	// SSEErrorRateLimit is returned when the org has too many concurrent requests in flight.
	// Clients should prompt the user to wait and retry.
	SSEErrorRateLimit SSEErrorCode = "rate_limit"
	// SSEErrorModelNotConfigured is returned when no enabled AI model is found for the org.
	// Clients should guide admins to AI Settings → Models to add or enable a model.
	SSEErrorModelNotConfigured SSEErrorCode = "model_not_configured"
	// SSEErrorInternal covers all other server-side errors. No specific action is available;
	// clients should show a generic retry prompt.
	SSEErrorInternal SSEErrorCode = "internal_error"
)

// SSEEvent is the JSON payload of a single Server-Sent Event frame.
// All fields are optional except Type; only the fields relevant to the
// specific event type are populated.
type SSEEvent struct {
	// Type identifies what kind of event this is.
	Type SSEEventType `json:"type"`
	// Content carries a text fragment (type=content) or human-readable error description (type=error).
	Content string `json:"content,omitempty"`
	// ErrorCode classifies the error so clients can render actionable guidance (type=error only).
	ErrorCode SSEErrorCode `json:"errorCode,omitempty"`
	// ConversationUID is set on the done event so the client can link future requests.
	ConversationUID string `json:"conversationUid,omitempty"`
	// Title is set on the done event for new conversations, carrying the auto-generated title.
	// Empty for existing conversations or when title generation fails/times out.
	Title string `json:"title,omitempty"`
	// ToolName is the name of the tool being invoked (type=tool_call) or whose result
	// is being delivered (type=tool_result).
	ToolName string `json:"toolName,omitempty"`
	// ToolCallID correlates tool_call and tool_result events for the same invocation.
	ToolCallID string `json:"toolCallId,omitempty"`
	// Arguments is the JSON-encoded arguments string sent to the tool (type=tool_call).
	Arguments string `json:"arguments,omitempty"`
	// Result is the tool's plain-text response (type=tool_result).
	Result string `json:"result,omitempty"`
}

// ModelRequest is the unified request body for POST /agent/models and
// PUT /agent/models/:uid. Required-field validation is enforced by the
// service layer (not gin binding) so that service calls made outside of
// HTTP handlers are also validated correctly.
type ModelRequest struct {
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	APIKey    string `json:"apiKey"`
	// IsDefault marks this model as the org default. On create it is applied
	// atomically after insert; on update it is stored directly.
	IsDefault   bool    `json:"isDefault"`
	BaseURL     string  `json:"baseUrl"`
	MaxTokens   int     `json:"maxTokens"`
	Temperature float64 `json:"temperature"`
	IsEnabled   bool    `json:"isEnabled"`
	Priority    int     `json:"priority"`
	CostWeight  int     `json:"costWeight"`
	Tags        string  `json:"tags"`
}

// SaveOrgModelConfigRequest is the body for PUT /agent/settings/model-routing.
type SaveOrgModelConfigRequest struct {
	SelectionMode string `json:"selectionMode" binding:"required,oneof=fixed auto"`
	AutoStrategy  string `json:"autoStrategy" binding:"omitempty,oneof=failover cost round_robin capability"`
}

// CreateMCPServerRequest is the body for POST /agent/settings/mcp.
type CreateMCPServerRequest struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	URL     string `json:"url"`
	Command string `json:"command"`
	Args    string `json:"args"`
	Enabled bool   `json:"enabled"`
}

// SearchConversationsRequest carries the pagination parameters for
// GET /agent/conversations.
type SearchConversationsRequest struct {
	Offset int `form:"offset"`
	Limit  int `form:"limit"`
}

// UpdateConversationTitleRequest is the body for PUT /agent/conversations/:uid/title.
type UpdateConversationTitleRequest struct {
	Title string `json:"title" binding:"required"`
}
