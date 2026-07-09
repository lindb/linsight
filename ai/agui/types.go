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

// Package agui defines the AG-UI protocol event and message types used for
// both live SSE streaming and conversation history serialisation. Using the
// same schema for both paths lets the frontend handle streaming and history
// restore with a single code path.
package agui

import (
	"github.com/cloudwego/eino/schema"
)

// EventType identifies the kind of AG-UI event frame.
type EventType string

const (
	// RunStarted signals the beginning of an agent run.
	RunStarted EventType = "RUN_STARTED"
	// RunFinished signals successful completion; carries conversationUid and title.
	RunFinished EventType = "RUN_FINISHED"
	// RunError signals an unrecoverable agent error; carries code and message.
	RunError EventType = "RUN_ERROR"

	// TextMessageStart opens a streaming text message; carries messageId and role.
	TextMessageStart EventType = "TEXT_MESSAGE_START"
	// TextMessageContent carries one streamed text delta; carries messageId and delta.
	TextMessageContent EventType = "TEXT_MESSAGE_CONTENT"
	// TextMessageEnd closes a streaming text message; carries messageId.
	TextMessageEnd EventType = "TEXT_MESSAGE_END"

	// ToolCallStart opens a tool invocation; carries toolCallId, toolCallName, parentMessageId.
	ToolCallStart EventType = "TOOL_CALL_START"
	// ToolCallArgs carries a streaming argument delta; carries toolCallId and delta.
	ToolCallArgs EventType = "TOOL_CALL_ARGS"
	// ToolCallEnd closes a tool invocation; carries toolCallId.
	ToolCallEnd EventType = "TOOL_CALL_END"
	// ToolCallResult delivers the tool response; carries messageId, toolCallId, toolCallName, content, and optional guidance.
	ToolCallResult EventType = "TOOL_CALL_RESULT"

	// MessagesSnapshot delivers the full history of a conversation.
	// Used by the GET /conversations/:uid/messages endpoint so the frontend
	// processes history with the same schema as live streaming events.
	MessagesSnapshot EventType = "MESSAGES_SNAPSHOT"

	// Reasoning phase events follow the AG-UI spec:
	// https://docs.ag-ui.com/concepts/reasoning
	// Reasoning and text messages are separate message lifecycles.
	ReasoningMessageStart   EventType = "REASONING_MESSAGE_START"
	ReasoningMessageContent EventType = "REASONING_MESSAGE_CONTENT"
	ReasoningMessageEnd     EventType = "REASONING_MESSAGE_END"

	// StepStarted opens a named execution step (one model-call iteration).
	// Fires before REASONING_MESSAGE_START so the frontend can group the full iteration
	// (reasoning → tool call(s) → tool result(s) → final text) under one step header.
	StepStarted EventType = "STEP_STARTED"
	// StepFinished closes a named execution step.
	// Fires after the last TOOL_CALL_RESULT in the iteration (or after model stream ends
	// for text-only iterations); carries the total iteration elapsed time.
	StepFinished EventType = "STEP_FINISHED"

	// UserMessage represents a complete, non-streaming user input event.
	// Used only in snapshot (GET /conversations/:uid/messages); never emitted during
	// live SSE because the frontend already renders the user's input optimistically.
	UserMessage EventType = "USER_MESSAGE"
)

// RunOutcome describes why a run ended. "complete" is the normal path; "interrupt" means
// the agent stopped to wait for human approval before executing one or more tools.
type RunOutcome string

const (
	// RunOutcomeComplete is the normal successful completion path.
	RunOutcomeComplete RunOutcome = "complete"
	// RunOutcomeInterrupt means the run paused waiting for human approval.
	RunOutcomeInterrupt RunOutcome = "interrupt"
)

// PendingToolCall describes a single tool invocation waiting for human approval.
// Sent inside InterruptPayload so the frontend can render an approval card with
// the correct input schema and echo back the tool call ID on resume.
type PendingToolCall struct {
	// ID is the LLM-assigned tool_call_id; echoed back in ResumePayload to correlate.
	ID string `json:"id"`
	// Name is the tool function name (e.g. "create_alert_rule").
	Name string `json:"name"`
	// Arguments is the JSON-encoded argument string the LLM produced.
	Arguments string `json:"arguments"`
	// CardType is the frontend component hint (e.g. "alert_rule_card").
	// Sourced from the "_gov_card_type" key in ToolInfo.Extra.
	CardType string `json:"cardType,omitempty"`
}

// InterruptPayload carries the pending tool calls that triggered the interrupt.
// Embedded in a RUN_FINISHED event (outcome=interrupt) so the frontend renders
// approval cards and echoes back interruptId when resuming.
type InterruptPayload struct {
	// InterruptID is the server-generated UUID for this interrupt instance.
	// The client must echo it back in ResumePayload.InterruptID.
	InterruptID string `json:"interruptId"`
	// ToolCalls lists the pending tool invocations waiting for approval.
	ToolCalls []PendingToolCall `json:"toolCalls"`
	// Status is the interrupt lifecycle state (pending/approved/rejected/expired).
	// Omitted on the live SSE frame (the interrupt is always pending at emit time);
	// populated on history replay so the frontend can render the approval card as
	// actionable (pending) or resolved (approved/rejected/expired).
	Status string `json:"status,omitempty"`
}

// ErrorCode classifies RUN_ERROR events so clients can show actionable guidance.
type ErrorCode string

const (
	// ErrorRateLimit is sent when the org has too many concurrent requests.
	ErrorRateLimit ErrorCode = "rate_limit"
	// ErrorModelNotConfigured is sent when no enabled model is found for the org.
	ErrorModelNotConfigured ErrorCode = "model_not_configured"
	// ErrorInternal covers all other server-side errors.
	ErrorInternal ErrorCode = "internal_error"
)

// MessageStatus describes the lifecycle state of a message.
// In SSE streaming, status transitions from processing → streaming → complete.
// In history restore, all messages carry StatusComplete.
type MessageStatus string

const (
	// StatusProcessing means the agent has started working (tool invoked, waiting for result).
	StatusProcessing MessageStatus = "processing"
	// StatusStreaming means the LLM is actively emitting text deltas.
	StatusStreaming MessageStatus = "streaming"
	// StatusComplete means the message is fully formed and persisted.
	StatusComplete MessageStatus = "complete"
	// StatusError means the message represents an agent-side failure.
	StatusError MessageStatus = "error"
)

// ToolCall is the AG-UI DTO representation of a single tool invocation.
// It intentionally does not embed eino's schema.ToolCall so that the frontend
// contract remains stable and can carry result fields alongside the Extra guidance.
type ToolCall struct {
	// ID is the tool_call_id assigned by the LLM.
	ID string `json:"id"`
	// Name is the tool function name.
	Name string `json:"name"`
	// Arguments is the JSON-encoded arguments string the LLM produced.
	Arguments string `json:"arguments,omitempty"`
	// Content is the tool execution result text.
	// Mirrors TOOL_CALL_RESULT.content; populated in snapshot, empty during live streaming.
	Content string `json:"content,omitempty"`
	// Status mirrors TOOL_CALL_RESULT.status (complete or error).
	// Populated in snapshot, empty during live streaming.
	Status MessageStatus `json:"status,omitempty"`
	// Extra carries optional render hints and metadata from the tool result.
	// Key conventions: "render_type" (string), "raw_data" (JSON), "icon" (string),
	// "datasource_type" (string for datasource_query). Nil when the tool returned plain text.
	Extra map[string]any `json:"extra,omitempty"`
}

// Event is the JSON envelope for all AG-UI event types.
// Fields not relevant to a specific event type are omitted (omitempty).
type Event struct {
	Type EventType `json:"type"`

	// RunStarted / RunFinished
	RunID           string `json:"runId,omitempty"`
	ConversationUID string `json:"conversationUid,omitempty"`

	// RunFinished (new conversation title)
	Title string `json:"title,omitempty"`

	// TextMessage* and ToolCall* events
	MessageID string `json:"id,omitempty"`
	// Status is the lifecycle state of the message referenced by MessageID.
	// Set on TEXT_MESSAGE_START (streaming), TEXT_MESSAGE_END (complete),
	// TOOL_CALL_START (processing), and TOOL_CALL_RESULT (complete).
	// Empty for events that do not change message state.
	Status MessageStatus   `json:"status,omitempty"`
	Role   schema.RoleType `json:"role,omitempty"`
	Delta  string          `json:"delta,omitempty"`

	// ToolCall* events
	ToolCallID      string `json:"toolCallId,omitempty"`
	ToolCallName    string `json:"toolCallName,omitempty"`
	ParentMessageID string `json:"parentMessageId,omitempty"`

	// ToolCallResult
	Content string `json:"content,omitempty"`

	// ToolCallResult optional structured render hint and metadata.
	// Mirrors ToolCall.Extra; populated after tool execution completes.
	Extra map[string]any `json:"extra,omitempty"`

	// RunError
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`

	// Step lifecycle: STEP_STARTED and STEP_FINISHED.
	// StepID is a short UUID distinct from ToolCallID; one per model-call iteration.
	// StepName carries a caller-supplied label (e.g. "Reasoning", "Deep Analysis").
	// Empty string is valid for unnamed steps.
	StepID   string `json:"stepId,omitempty"`
	StepName string `json:"stepName,omitempty"`

	// ElapsedMs is the processing duration in milliseconds.
	// Set on TEXT_MESSAGE_END, REASONING_MESSAGE_END, TOOL_CALL_RESULT, and STEP_FINISHED.
	ElapsedMs int64 `json:"elapsedMs,omitempty"`

	// Sequence is the run-scoped insertion order assigned by the server.
	// Set on the first event of each DB row (START / single-event rows).
	// Used by the frontend for ordering and deduplication in snapshot replay.
	Sequence int `json:"sequence,omitempty"`

	// MessagesSnapshot (legacy — deprecated, use Events instead).
	// Kept for backwards compatibility; will be removed in a future release.
	// TODO: remove it?
	Messages []Message `json:"messages,omitempty"`

	// Events carries the ordered list of agui.Event entries for a MESSAGES_SNAPSHOT response.
	// Each entry is a complete event with Content (not Delta) for text events, so the
	// frontend can replay the full conversation using the same event-processing path as
	// live streaming.
	Events []Event `json:"events,omitempty"`

	// Outcome describes why the run ended. Empty string is equivalent to RunOutcomeComplete
	// (backwards compatible). Set to RunOutcomeInterrupt when the agent paused for approval.
	Outcome RunOutcome `json:"outcome,omitempty"`
	// Interrupt carries the pending tool calls when Outcome == RunOutcomeInterrupt.
	// Nil for normal RUN_FINISHED events.
	Interrupt *InterruptPayload `json:"interrupt,omitempty"`
}

// Message is the unified representation of a single conversation turn.
// It is used both as SSE event payload and as the messages array in
// MessagesSnapshot events, so the frontend uses one code path for both
// streaming and history restore.
//
// Role uses schema.RoleType directly; the custom "error" role is defined
// as agentmodel.RoleError in ai/model to keep role constants co-located
// with the DB model that stores them.
type Message struct {
	// ID is the AG-UI message identifier (MessageUID from the DB row).
	ID string `json:"id"`
	// Role is the message sender (user, assistant, error).
	// Tool messages are never included — their guidance is merged into the
	// parent assistant message's ToolCalls[x].Extra.
	Role schema.RoleType `json:"role"`
	// Status describes the message lifecycle state.
	// All messages in a MessagesSnapshot carry StatusComplete.
	Status MessageStatus `json:"status,omitempty"`
	// Content is the full accumulated text for completed messages.
	Content string `json:"content,omitempty"`
	// ReasoningContent is the accumulated reasoning/thinking text for completed messages.
	// Non-empty only when the LLM supports extended thinking (e.g. DeepSeek R1, Claude w/ thinking).
	// Stripped before sending messages to LLM API to avoid 400 errors.
	ReasoningContent string `json:"reasoningContent,omitempty"`
	// Delta carries incremental text during StatusStreaming; empty otherwise.
	Delta string `json:"delta,omitempty"`
	// ToolCalls lists tool invocations requested by an assistant message.
	// Each entry may carry an Extra *Guidance when the tool returned structured data.
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	// ToolCallID correlates a tool-result message with its assistant tool_call.
	// Non-empty only when Role == schema.Tool; kept for internal use but typically
	// filtered out before sending to the frontend.
	ToolCallID string `json:"toolCallId,omitempty"`
	// Name is the tool name for role=tool messages.
	Name string `json:"name,omitempty"`
	// TurnUID groups all messages (user + assistant) that belong to the same agent turn.
	// Clients can use this to correlate a user message with its assistant reply and
	// to implement turn-level operations such as retry or feedback.
	TurnUID string `json:"turnUid,omitempty"`
	// CreatedAt is the Unix millisecond timestamp when the message was persisted.
	CreatedAt int64 `json:"createdAt,omitempty"`
}
