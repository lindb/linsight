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

package service

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/schema"

	"github.com/lindb/linsight/ai/agui"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ConversationService manages conversations and their messages.
type ConversationService interface {
	// GetConversation returns the conversation with the given UID owned by the current user in ctx.
	// Returns an error if no matching conversation exists or the user does not own it.
	GetConversation(ctx context.Context, uid string) (*agentmodel.Conversation, error)
	// CreateConversation creates a new conversation for the user in ctx and returns it.
	CreateConversation(ctx context.Context) (*agentmodel.Conversation, error)
	// ListConversations returns paginated conversations for the user in ctx, newest first.
	ListConversations(ctx context.Context, offset, limit int) ([]*agentmodel.Conversation, int64, error)
	// DeleteConversation removes a conversation and all its messages in a transaction.
	DeleteConversation(ctx context.Context, uid string) error
	// UpdateConversationTitle sets the title of an existing conversation.
	UpdateConversationTitle(ctx context.Context, convID int64, title string) error
	// GetConversationMessages returns up to limit messages for the conversation identified
	// by uid (org-isolated), ordered oldest-first. Used by the history restore API.
	GetConversationMessages(ctx context.Context, uid string, limit int) ([]agui.Message, error)
	// GetRecentMessages returns up to limit messages for the conversation ordered
	// oldest-first. Used by the agent to build LLM context.
	GetRecentMessages(ctx context.Context, conversationID int64, limit int) ([]*agentmodel.Message, error)
	// SaveMessage persists a single message to the conversation.
	SaveMessage(ctx context.Context, msg *agentmodel.Message) error

	// SaveEvent persists a completed AG-UI event to the events table.
	SaveEvent(ctx context.Context, evt *agentmodel.Event) error
	// GetConversationEvents returns the ordered events for a conversation as a flat
	// []agui.Event slice suitable for snapshot delivery. Each DB event row is expanded
	// into the corresponding Start+Content+End triple (or tool series) so the frontend
	// can replay the history using the same event-processing path as live streaming.
	// Text and reasoning CONTENT events carry Content (full text), not Delta.
	GetConversationEvents(ctx context.Context, uid string, limit int) ([]agui.Event, error)
	// GetRecentEvents returns event rows for LLM context building, ordered oldest-first.
	// Reasoning and step events are excluded at the query level so callers never need to
	// filter them. Error-status rows are also excluded.
	GetRecentEvents(ctx context.Context, convUID string, limit int) ([]*agentmodel.Event, error)
}

type conversationService struct {
	db dbpkg.DB
}

// NewConversationService creates an ConversationService backed by db.
func NewConversationService(db dbpkg.DB) ConversationService {
	return &conversationService{db: db}
}

// GetConversation returns the conversation with the given UID owned by the current user in ctx.
// Returns an error if no matching conversation exists or the user does not own it.
func (s *conversationService) GetConversation(ctx context.Context, uid string) (*agentmodel.Conversation, error) {
	user := util.GetUser(ctx)
	var conv agentmodel.Conversation
	if err := s.db.Get(&conv, "org_id = ? and user_id = ? and uid = ?", user.Org.ID, user.User.ID, uid); err != nil {
		return nil, err
	}
	return &conv, nil
}

// CreateConversation creates a new conversation for the user in ctx and returns it.
func (s *conversationService) CreateConversation(ctx context.Context) (*agentmodel.Conversation, error) {
	user := util.GetUser(ctx)
	conv := &agentmodel.Conversation{
		UID:    uuid.GenerateShortUUID(),
		OrgID:  user.Org.ID,
		UserID: user.User.ID,
	}
	conv.CreatedBy = user.User.ID
	conv.UpdatedBy = user.User.ID
	if err := s.db.Create(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// ListConversations returns paginated conversations for the user in ctx, newest first.
func (s *conversationService) ListConversations(
	ctx context.Context, offset, limit int,
) ([]*agentmodel.Conversation, int64, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID
	total, err := s.db.Count(&agentmodel.Conversation{}, "org_id = ? and user_id = ?", orgID, userID)
	if err != nil {
		return nil, 0, err
	}
	var convs []*agentmodel.Conversation
	if err := s.db.FindForPaging(&convs, offset, limit, "id desc",
		"org_id = ? and user_id = ?", orgID, userID); err != nil {
		return nil, 0, err
	}
	return convs, total, nil
}

// DeleteConversation removes a conversation and all its messages in a transaction.
func (s *conversationService) DeleteConversation(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID
	return s.db.Transaction(func(tx dbpkg.DB) error {
		// Resolve the conversation so we can delete messages by conversation_id.
		// Enforce user ownership to prevent cross-user deletion within the same org.
		var conv agentmodel.Conversation
		if err := tx.Get(&conv, "org_id = ? and user_id = ? and uid = ?", orgID, userID, uid); err != nil {
			return err
		}
		// Delete all messages in this conversation first (no FK constraint assumed).
		if err := tx.Delete(&agentmodel.Message{}, "conversation_id = ?", conv.ID); err != nil {
			return err
		}
		// Delete all events for this conversation.
		if err := tx.Delete(&agentmodel.Event{}, "conv_uid = ?", conv.UID); err != nil {
			return err
		}
		// Delete all feedback records for this conversation (no FK constraint assumed).
		if err := tx.Delete(&agentmodel.MessageFeedback{}, "conversation_id = ?", conv.ID); err != nil {
			return err
		}
		return tx.Delete(&agentmodel.Conversation{}, "id = ?", conv.ID)
	})
}

// UpdateConversationTitle sets the title of an existing conversation.
func (s *conversationService) UpdateConversationTitle(ctx context.Context, convID int64, title string) error {
	user := util.GetUser(ctx)
	return s.db.UpdateSingle(&agentmodel.Conversation{}, "title", title,
		"id = ? and org_id = ? and user_id = ?", convID, user.Org.ID, user.User.ID)
}

// GetConversationMessages returns up to limit messages for the conversation
// identified by uid, org-isolated, ordered oldest-first. Used by the history restore API.
//
// The function runs three passes:
//  1. Index tool-result rows by tool_call_id — each entry carries content, status, and guidance.
//  2. Group non-tool rows by turn_uid (falling back to message_uid for pre-migration rows) to
//     reconstruct the logical turns in insertion order.
//  3. Assemble one or two agui.Message values per turn: a user message and an assistant message
//     that carries all tool calls (with their results) and the final text content.
//
// Tool messages are never included in the output; their content is merged into
// ToolCalls[x].Content/Status/Extra so the frontend uses a single code path for
// both streaming and history restore.
func (s *conversationService) GetConversationMessages(ctx context.Context, uid string, limit int) ([]agui.Message, error) {
	conv, err := s.GetConversation(ctx, uid)
	if err != nil {
		return nil, err
	}
	var msgs []*agentmodel.Message
	if err := s.db.FindForPaging(&msgs, 0, limit, "id asc",
		"conversation_id = ?", conv.ID); err != nil {
		return nil, err
	}

	// Pass 1: index tool-result rows by tool_call_id.
	type toolResult struct {
		content string
		status  agui.MessageStatus
		extra   map[string]any
	}
	toolResultMap := make(map[string]*toolResult, len(msgs))
	for _, m := range msgs {
		if m.Role != schema.Tool || m.ToolCallID == "" {
			continue
		}
		var sm schema.Message
		if json.Unmarshal(m.RawMessage, &sm) != nil {
			continue
		}
		res := &toolResult{content: sm.Content, status: agui.StatusComplete}
		if m.Status == string(agui.StatusError) {
			res.status = agui.StatusError
		}
		if len(m.Extra) > 0 {
			var extra map[string]any
			if json.Unmarshal(m.Extra, &extra) == nil {
				res.extra = extra
			}
		}
		toolResultMap[m.ToolCallID] = res
	}

	// Pass 2: group non-tool rows by turn_uid, preserving insertion order.
	// Pre-migration rows with empty TurnUID form singleton groups keyed by MessageUID.
	turnOrder := make([]string, 0, len(msgs))
	turnMsgs := make(map[string][]*agentmodel.Message, len(msgs))
	for _, m := range msgs {
		if m.Role == schema.Tool {
			continue
		}
		key := m.TurnUID
		if key == "" {
			key = m.MessageUID
		}
		if _, ok := turnMsgs[key]; !ok {
			turnOrder = append(turnOrder, key)
		}
		turnMsgs[key] = append(turnMsgs[key], m)
	}

	// Pass 3: assemble agui.Message values for each turn.
	result := make([]agui.Message, 0, len(msgs))
	for _, key := range turnOrder {
		rows := turnMsgs[key]

		var userMsg *agentmodel.Message
		var errMsg *agentmodel.Message
		var finalMsg *agentmodel.Message            // last assistant row with non-empty Content
		var finalMsgContent string                  // Content of finalMsg, cached to avoid re-unmarshal
		var finalMsgReasoningContent string         // ReasoningContent of finalMsg, cached alongside Content
		var firstTCMsg *agentmodel.Message          // first assistant row with ToolCalls (for interrupted-turn ID)
		var toolCalls []agui.ToolCall
		seenCallIDs := make(map[string]bool)

		for _, m := range rows {
			var sm schema.Message
			if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
				continue
			}
			switch m.Role {
			case schema.User:
				userMsg = m
			case schema.Assistant:
				if m.Status == string(agui.StatusError) {
					errMsg = m
					continue
				}
				// Accumulate tool calls, merging results from Pass 1.
				if len(sm.ToolCalls) > 0 {
					if firstTCMsg == nil {
						firstTCMsg = m
					}
					for _, tc := range sm.ToolCalls {
						if seenCallIDs[tc.ID] {
							continue
						}
						seenCallIDs[tc.ID] = true
						call := agui.ToolCall{
							ID:        tc.ID,
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						}
						if res, ok := toolResultMap[tc.ID]; ok {
							call.Content = res.content
							call.Status = res.status
							call.Extra = res.extra
						}
						toolCalls = append(toolCalls, call)
					}
				}
				// A row may carry both ToolCalls and Content (written by
				// savePartialAssistantMessage at the end of a tool-call chain).
				if sm.Content != "" {
					finalMsg = m
					finalMsgContent = sm.Content
					finalMsgReasoningContent = sm.ReasoningContent
				}
			}
		}

		// Emit user message first.
		if userMsg != nil {
			var sm schema.Message
			if err := json.Unmarshal(userMsg.RawMessage, &sm); err == nil {
				result = append(result, agui.Message{
					ID:        userMsg.MessageUID,
					TurnUID:   key,
					Role:      sm.Role,
					Content:   sm.Content,
					Status:    agui.StatusComplete,
					CreatedAt: userMsg.CreatedAt,
				})
			}
		}

		// Error turn: emit up to two messages to mirror the SSE stream structure.
		//
		// If partial text was accumulated before the error (finalMsg != nil with non-empty
		// content), emit it as a complete assistant message first — this reconstructs the
		// TEXT_MESSAGE_* bubble the client already showed during live streaming. The tool
		// calls belong to this message (they were streamed as children of the same bubble).
		//
		// Always follow with the error message (status=error, content=error description).
		// When there is no partial text, attach tool calls to the error message so they
		// are still visible (pure-tool-error path, current behavior unchanged).
		if errMsg != nil {
			partialEmitted := false
			if finalMsg != nil && finalMsgContent != "" {
				result = append(result, agui.Message{
					ID:               finalMsg.MessageUID,
					TurnUID:          key,
					Role:             schema.Assistant,
					Status:           agui.StatusComplete,
					Content:          finalMsgContent,
					ReasoningContent: finalMsgReasoningContent,
					ToolCalls:        toolCalls,
					CreatedAt:        finalMsg.CreatedAt,
				})
				partialEmitted = true
			}
			var errSm schema.Message
			_ = json.Unmarshal(errMsg.RawMessage, &errSm)
			// errSm.Content is "" on unmarshal failure — zero value is the correct fallback.
			errOut := agui.Message{
				ID:        errMsg.MessageUID,
				TurnUID:   key,
				Role:      schema.Assistant,
				Status:    agui.StatusError,
				Content:   errSm.Content,
				CreatedAt: errMsg.CreatedAt,
			}
			// Attach tool calls to the error message only when no partial text
			// message was emitted — otherwise they are already shown in Message 1.
			if !partialEmitted {
				errOut.ToolCalls = toolCalls
			}
			result = append(result, errOut)
			continue
		}

		// Normal turn: emit assistant message with tool calls and/or final content.
		if finalMsg != nil || len(toolCalls) > 0 {
			out := agui.Message{
				Role:      schema.Assistant,
				TurnUID:   key,
				Status:    agui.StatusComplete,
				ToolCalls: toolCalls,
			}
			if finalMsg != nil {
				// Reuse the cached content instead of re-unmarshalling finalMsg.RawMessage.
				out.ID = finalMsg.MessageUID
				out.Content = finalMsgContent
				out.ReasoningContent = finalMsgReasoningContent
				out.CreatedAt = finalMsg.CreatedAt
			} else {
				// Interrupted turn: no final content row; use the first tool-call
				// assistant row so the ID refers to a real DB row (needed for feedback lookup).
				out.ID = firstTCMsg.MessageUID
				out.CreatedAt = firstTCMsg.CreatedAt
			}
			result = append(result, out)
		}
	}
	return result, nil
}

// GetRecentMessages returns up to limit messages for the conversation, ordered
// by ID descending (newest first), then reversed so the oldest is at index 0.
func (s *conversationService) GetRecentMessages(ctx context.Context, conversationID int64, limit int) ([]*agentmodel.Message, error) {
	var msgs []*agentmodel.Message
	if err := s.db.FindForPaging(&msgs, 0, limit, "id desc",
		"conversation_id = ?", conversationID); err != nil {
		return nil, err
	}
	// Reverse so messages are ordered oldest-first for LLM context assembly.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// SaveMessage persists a single message to the conversation.
func (s *conversationService) SaveMessage(ctx context.Context, msg *agentmodel.Message) error {
	return s.db.Create(msg)
}

// SaveEvent persists a completed AG-UI event to the events table.
func (s *conversationService) SaveEvent(_ context.Context, evt *agentmodel.Event) error {
	return s.db.Create(evt)
}

// GetConversationEvents returns the ordered AG-UI events for a conversation,
// expanding each DB event row into the corresponding SSE event triple (or series)
// so the frontend can replay history using the same event-processing path as live streaming.
//
// Text and reasoning CONTENT events carry Content (full text), not Delta, so the frontend
// can distinguish snapshot full-content from streaming incremental deltas.
//
// Ownership is verified via GetConversation; the events table has no org_id/user_id columns.
func (s *conversationService) GetConversationEvents(ctx context.Context, uid string, limit int) ([]agui.Event, error) {
	conv, err := s.GetConversation(ctx, uid)
	if err != nil {
		return nil, err
	}
	var rows []*agentmodel.Event
	// Order by primary key (global insert order) instead of sequence, because
	// sequence resets to 1 at the start of every turn and cannot be used for
	// cross-turn global ordering.
	if err := s.db.FindForPaging(&rows, 0, limit, "id asc",
		"conv_uid = ?", conv.UID); err != nil {
		return nil, err
	}

	// First pass: build run_id → assistant text messageUID map so tool_call rows can
	// reference the correct parentMessageId when emitting TOOL_CALL_START.
	assistantMsgByRun := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Type == agentmodel.EventTypeAssistantText {
			assistantMsgByRun[r.RunID] = r.MessageUID
		}
	}

	// Second pass: expand each DB row to one or more agui.Event values.
	result := make([]agui.Event, 0, len(rows)*2)
	// stepStack tracks per-step UUIDs for STEP_STARTED/STEP_FINISHED pairing.
	// Steps do not nest (each tool call is fully sequential), so a stack is safe.
	stepStack := make([]string, 0, 4)

	for _, r := range rows {
		switch r.Type {
		case agentmodel.EventTypeUserMessage:
			// User messages are never streamed; use a single flat USER_MESSAGE event
			// with StatusComplete so the frontend does not apply streaming semantics.
			result = append(result, agui.Event{
				Type:      agui.UserMessage,
				MessageID: r.MessageUID,
				Role:      schema.User,
				Content:   r.Content,
				Status:    agui.StatusComplete,
				RunID:     r.RunID,
				Sequence:  r.Sequence,
			})

		case agentmodel.EventTypeAssistantText:
			result = append(result,
				agui.Event{Type: agui.TextMessageStart, MessageID: r.MessageUID, Role: schema.Assistant, Status: agui.StatusComplete, RunID: r.RunID, Sequence: r.Sequence},
				agui.Event{Type: agui.TextMessageContent, MessageID: r.MessageUID, Delta: r.Content},
				agui.Event{Type: agui.TextMessageEnd, MessageID: r.MessageUID, Status: agui.StatusComplete, ElapsedMs: r.ElapsedMs},
			)

		case agentmodel.EventTypeReasoning:
			result = append(result,
				agui.Event{Type: agui.ReasoningMessageStart, MessageID: r.MessageUID, Status: agui.StatusComplete, RunID: r.RunID, Sequence: r.Sequence},
				agui.Event{Type: agui.ReasoningMessageContent, MessageID: r.MessageUID, Delta: r.Content},
				agui.Event{Type: agui.ReasoningMessageEnd, MessageID: r.MessageUID, Status: agui.StatusComplete, ElapsedMs: r.ElapsedMs},
			)

		case agentmodel.EventTypeToolCall:
			// Each tool_call row stores a single-element ToolCalls slice in RawMessage.
			var sm schema.Message
			if err := json.Unmarshal(r.RawMessage, &sm); err != nil || len(sm.ToolCalls) == 0 {
				continue
			}
			tc := sm.ToolCalls[0]
			parentMsgUID := assistantMsgByRun[r.RunID]
			// Read static tool metadata (icon/render_type) stored at call time for pre-rendering.
			var extra map[string]any
			if len(r.Extra) > 0 {
				_ = json.Unmarshal(r.Extra, &extra)
			}
			result = append(result,
				agui.Event{Type: agui.ToolCallStart, ToolCallID: tc.ID, ToolCallName: tc.Function.Name, ParentMessageID: parentMsgUID, Status: agui.StatusProcessing, Extra: extra, RunID: r.RunID, Sequence: r.Sequence},
				agui.Event{Type: agui.ToolCallArgs, ToolCallID: tc.ID, Delta: tc.Function.Arguments},
				agui.Event{Type: agui.ToolCallEnd, ToolCallID: tc.ID},
			)

		case agentmodel.EventTypeToolResult:
			var extra map[string]any
			if len(r.Extra) > 0 {
				var m map[string]any
				if json.Unmarshal(r.Extra, &m) == nil {
					extra = m
				}
			}
			status := agui.StatusComplete
			if r.Status == string(agui.StatusError) {
				status = agui.StatusError
			}
			result = append(result, agui.Event{
				Type:         agui.ToolCallResult,
				MessageID:    r.MessageUID,
				ToolCallID:   r.ToolCallID,
				ToolCallName: r.ToolName,
				Content:      r.Content,
				Extra:        extra,
				Status:       status,
				RunID:        r.RunID,
				ElapsedMs:    r.ElapsedMs,
				Sequence:     r.Sequence,
			})

		case agentmodel.EventTypeStepStarted:
			// Generate a fresh per-step UUID for SSE STEP_STARTED/STEP_FINISHED correlation.
			// The DB step_id column stores the model-call iteration ID (not the SSE step ID),
			// so we cannot reuse it here.
			stepUID := uuid.GenerateShortUUID()
			stepStack = append(stepStack, stepUID)
			result = append(result, agui.Event{
				Type:     agui.StepStarted,
				StepID:   stepUID,
				StepName: r.ToolName, // empty for iteration-level steps; non-empty legacy tool steps
				RunID:    r.RunID,
				Sequence: r.Sequence,
			})

		case agentmodel.EventTypeStepFinished:
			// Pop the matching STEP_STARTED UUID so STEP_FINISHED carries the same stepId.
			if len(stepStack) == 0 {
				// step_started was missing (e.g. write failure); skip to avoid unmatched event.
				continue
			}
			stepUID := stepStack[len(stepStack)-1]
			stepStack = stepStack[:len(stepStack)-1]
			result = append(result, agui.Event{
				Type:      agui.StepFinished,
				StepID:    stepUID,
				ElapsedMs: r.ElapsedMs,
				Sequence:  r.Sequence,
			})

		case agentmodel.EventTypeRunError:
			result = append(result, agui.Event{
				Type:     agui.RunError,
				Code:     r.Code,
				Message:  r.Content,
				RunID:    r.RunID,
				Sequence: r.Sequence,
			})
		}
	}
	return result, nil
}

// GetRecentEvents returns event rows for LLM context building, ordered oldest-first.
// Reasoning, step_started, and step_finished rows are excluded at the query level
// (models reject their own reasoning; step events carry no LLM-relevant content).
// Error-status rows are also excluded to prevent poisoning the context.
func (s *conversationService) GetRecentEvents(ctx context.Context, convUID string, limit int) ([]*agentmodel.Event, error) {
	var evts []*agentmodel.Event
	excludeTypes := []string{
		agentmodel.EventTypeReasoning,
		agentmodel.EventTypeStepStarted,
		agentmodel.EventTypeStepFinished,
	}
	if err := s.db.FindForPaging(&evts, 0, limit, "id desc",
		"conv_uid = ? AND type NOT IN ? AND status != ?",
		convUID, excludeTypes, string(agui.StatusError)); err != nil {
		return nil, err
	}
	// Reverse so events are ordered oldest-first for LLM context assembly.
	for i, j := 0, len(evts)-1; i < j; i, j = i+1, j-1 {
		evts[i], evts[j] = evts[j], evts[i]
	}
	return evts, nil
}
