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
		content  string
		status   agui.MessageStatus
		guidance *agui.Guidance
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
		if len(m.Guidance) > 0 {
			var g agui.Guidance
			if json.Unmarshal(m.Guidance, &g) == nil {
				res.guidance = &g
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
		var finalMsg *agentmodel.Message    // last assistant row with non-empty Content
		var finalMsgContent string          // Content of finalMsg, cached to avoid re-unmarshal
		var firstTCMsg *agentmodel.Message  // first assistant row with ToolCalls (for interrupted-turn ID)
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
							call.Extra = res.guidance
						}
						toolCalls = append(toolCalls, call)
					}
				}
				// A row may carry both ToolCalls and Content (written by
				// savePartialAssistantMessage at the end of a tool-call chain).
				if sm.Content != "" {
					finalMsg = m
					finalMsgContent = sm.Content
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
					ID:        finalMsg.MessageUID,
					TurnUID:   key,
					Role:      schema.Assistant,
					Status:    agui.StatusComplete,
					Content:   finalMsgContent,
					ToolCalls: toolCalls,
					CreatedAt: finalMsg.CreatedAt,
				})
				partialEmitted = true
			}
			var errSm schema.Message
			_ = json.Unmarshal(errMsg.RawMessage, &errSm)
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
				var sm schema.Message
				if err := json.Unmarshal(finalMsg.RawMessage, &sm); err == nil {
					out.ID = finalMsg.MessageUID
					out.Content = sm.Content
					out.CreatedAt = finalMsg.CreatedAt
				}
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
