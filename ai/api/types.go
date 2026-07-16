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

import (
	"errors"

	"github.com/lindb/linsight/ai/agui"
)

// ErrTooManyRequests is returned when the per-org concurrency limit is exceeded.
// Defined here (rather than in the service layer) so that ai/util can reference it
// without creating a circular import (ai/util ← ai/api; ai/service → ai/util).
var ErrTooManyRequests = errors.New("too many concurrent requests for this org")

// ChatRequest is the request body for POST /agent/chat and
// POST /agent/conversations/:uid/chat.
type ChatRequest struct {
	// Content is the user's plain-text message. Required on the initial turn;
	// may be empty on resume turns (where Resume carries all the necessary state).
	Content string `json:"content"`
	// Resume carries the approval decision when the client resumes an interrupted run.
	// Mutually exclusive with Content (service layer validates the combination).
	Resume *ResumePayload `json:"resume,omitempty"`
	// AutoApprove, when true, skips approval for tools that have _gov_need_approval=true
	// in this single request only. Request-level (not conversation-level) so that each
	// turn that bypasses governance is an explicit, auditable opt-in.
	AutoApprove bool `json:"autoApprove,omitempty"`
}

// ResumePayload carries the client's decision for a previously interrupted run.
// The client echoes back the interruptId it received in the RUN_FINISHED event.
type ResumePayload struct {
	// InterruptID is the server-generated UUID from InterruptPayload.InterruptID.
	InterruptID string `json:"interruptId" binding:"required"`
	// Approved is true if the user approved the pending tool invocations.
	Approved bool `json:"approved"`
	// EditedArgs is an optional map[toolCallID]json.RawMessage of edited arguments.
	// When present, the agent will override the LLM-generated arguments with these values
	// before executing the tool. Ignored when Approved is false.
	EditedArgs map[string]any `json:"editedArgs,omitempty"`
	// Comment is optional free-text from the reviewer (stored in InterruptRecord.Decision).
	Comment string `json:"comment,omitempty"`
}

// SSEErrorCode categorises the error so clients can show actionable guidance
// rather than a generic error message. Values are sent in RUN_ERROR events as the
// code field, matching agui.Event.Code.
type SSEErrorCode = agui.ErrorCode

const (
	// SSEErrorRateLimit is returned when the org has too many concurrent requests in flight.
	// Clients should prompt the user to wait and retry.
	SSEErrorRateLimit SSEErrorCode = agui.ErrorRateLimit
	// SSEErrorModelNotConfigured is returned when no enabled AI model is found for the org.
	// Clients should guide admins to AI Settings → Models to add or enable a model.
	SSEErrorModelNotConfigured SSEErrorCode = agui.ErrorModelNotConfigured
	// SSEErrorInternal covers all other server-side errors. No specific action is available;
	// clients should show a generic retry prompt.
	SSEErrorInternal SSEErrorCode = agui.ErrorInternal
)

// ModelRequest is the unified request body for POST /agent/models and
// PUT /agent/models/:uid. Required-field validation is enforced by the
// service layer (not gin binding) so that service calls made outside of
// HTTP handlers are also validated correctly.
type ModelRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
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

// RuleRequest is the unified body for POST /ai/settings/rules and
// PUT /ai/settings/rules/:uid. The Content field is a full Markdown document whose
// optional YAML frontmatter carries trigger_tools / trigger_keywords / variables;
// the API layer stores it verbatim and never parses it. Creating or editing a global
// rule (IsGlobal=true) requires super-admin (RoleLin); the service layer enforces this.
type RuleRequest struct {
	// RuleName is the human-readable unique name within the rule's scope. Required.
	RuleName string `json:"ruleName"`
	// Content is the full Markdown rule body (with optional frontmatter). Required.
	Content string `json:"content"`
	// IsGlobal marks the rule as applying to all orgs (org_id=0). Super-admin only.
	IsGlobal bool `json:"isGlobal"`
	// Priority orders rules when multiple are matched and concatenated (asc).
	Priority int `json:"priority"`
	// IsEnabled gates whether the rule is loaded into the runtime cache.
	IsEnabled bool `json:"isEnabled"`
}

// SkillRequest is the unified body for POST /ai/settings/skills and
// PUT /ai/settings/skills/:uid. Required-field validation is enforced by the
// service layer (not gin binding) so that service calls made outside of
// HTTP handlers are also validated correctly.
// Description, context_mode, and agent are carried in the Content YAML frontmatter.
type SkillRequest struct {
	// Name is the canonical skill name used as the tool parameter value. Required.
	Name string `json:"name"`
	// Content is the full Markdown skill document (YAML frontmatter + body). Required.
	Content string `json:"content"`
	// IsGlobal marks the skill as available to all orgs. Super-admin only.
	IsGlobal bool `json:"isGlobal"`
	// IsEnabled gates whether the skill is active.
	IsEnabled bool `json:"isEnabled"`
}

// AgentRequest is the unified body for POST /ai/settings/agents and
// PUT /ai/settings/agents/:uid. It configures a Specialist (sub-agent) as a
// Claude-Code-style agent.md document. Required-field validation is enforced by the
// service layer (not gin binding) so that service calls made outside of HTTP handlers
// are also validated correctly. The description / model / tools metadata is carried in
// the Content YAML frontmatter; the body is the specialist's system prompt.
type AgentRequest struct {
	// Name is the canonical specialist name and the suffix of its handoff tool
	// (handoff_to_<name>). Required; must be a tool-name-safe slug.
	Name string `json:"name"`
	// Content is the full agent.md Markdown document (YAML frontmatter + body). Required.
	Content string `json:"content"`
	// IsGlobal marks the specialist as available to all orgs. Super-admin only.
	IsGlobal bool `json:"isGlobal"`
	// IsEnabled gates whether the specialist is active (its handoff tool bound).
	IsEnabled bool `json:"isEnabled"`
}

// MemoryRequest is the unified body for POST /ai/settings/memories and
// PUT /ai/settings/memories/:uid. Content is required; Category is optional.
type MemoryRequest struct {
	// Content is the memory text. Required.
	Content string `json:"content"`
	// Category is an optional tag for grouping related memories.
	Category string `json:"category"`
	// IsEnabled gates whether the memory is active and loaded into the prompt.
	IsEnabled bool `json:"isEnabled"`
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

// FeedbackRequest is the body for PUT /ai/conversations/:uid/messages/:messageUID/feedback.
type FeedbackRequest struct {
	// Rating is required. 1 = thumbs up, -1 = thumbs down.
	Rating int8 `json:"rating" binding:"required,oneof=1 -1"`

	// Reasons is the optional list of predefined reason tags.
	// Each tag is validated against agentmodel.ValidFeedbackReasons in the service layer.
	Reasons []string `json:"reasons"`

	// Comment is free-text qualitative feedback from the user. Optional.
	Comment string `json:"comment"`

	// CorrectedContent is the user's manual correction of the assistant message.
	// Used as gold-label annotation for SFT/DPO fine-tuning pipelines. Optional.
	CorrectedContent string `json:"correctedContent"`
}
