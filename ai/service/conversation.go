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
	// GetConversation returns the conversation with the given UID for the org in ctx.
	// Returns an error if no matching conversation exists.
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

// GetConversation returns the conversation with the given UID for the org in ctx.
// Returns an error if no matching conversation exists.
func (s *conversationService) GetConversation(ctx context.Context, uid string) (*agentmodel.Conversation, error) {
	user := util.GetUser(ctx)
	var conv agentmodel.Conversation
	if err := s.db.Get(&conv, "org_id = ? and uid = ?", user.Org.ID, uid); err != nil {
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
	return s.db.Transaction(func(tx dbpkg.DB) error {
		// Resolve the conversation so we can delete messages by conversation_id.
		var conv agentmodel.Conversation
		if err := tx.Get(&conv, "org_id = ? and uid = ?", orgID, uid); err != nil {
			return err
		}
		// Delete all messages in this conversation first (no FK constraint assumed).
		if err := tx.Delete(&agentmodel.Message{}, "conversation_id = ?", conv.ID); err != nil {
			return err
		}
		return tx.Delete(&agentmodel.Conversation{}, "id = ?", conv.ID)
	})
}

// UpdateConversationTitle sets the title of an existing conversation.
func (s *conversationService) UpdateConversationTitle(ctx context.Context, convID int64, title string) error {
	return s.db.UpdateSingle(&agentmodel.Conversation{}, "title", title, "id = ?", convID)
}

// GetConversationMessages returns up to limit messages for the conversation
// identified by uid, org-isolated, ordered oldest-first. Used by the history restore API.
// Tool messages are filtered out; their Guidance is merged into the parent assistant
// message's ToolCalls[x].Extra so the frontend uses a single code path for both
// streaming and history restore.
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

	// First pass: build guidanceMap from tool-result rows using the physical
	// tool_call_id and guidance columns — no JSON parsing needed here.
	guidanceMap := make(map[string]*agui.Guidance, len(msgs))
	for _, m := range msgs {
		if m.Role != schema.Tool || m.ToolCallID == "" || len(m.Guidance) == 0 {
			continue
		}
		var g agui.Guidance
		if json.Unmarshal(m.Guidance, &g) == nil {
			guidanceMap[m.ToolCallID] = &g
		}
	}

	// Second pass: group non-tool rows by TurnUID into ordered turn groups.
	// For old messages where TurnUID is empty, fall back to MessageUID so each
	// row forms its own singleton group — preserving pre-migration behaviour.
	type turnGroup struct {
		userMsg      *agentmodel.Message
		toolCalls    []agui.ToolCall
		firstTCMsgUID string              // MessageUID of the first intermediate assistant row
		finalMsg     *agentmodel.Message // last assistant row that carries Content
		errorMsg     *agentmodel.Message
	}
	groupOrder := make([]string, 0, len(msgs))
	groups := make(map[string]*turnGroup, len(msgs))

	for _, m := range msgs {
		if m.Role == schema.Tool {
			continue
		}
		var sm schema.Message
		if err := json.Unmarshal(m.RawMessage, &sm); err != nil {
			continue
		}

		key := m.TurnUID
		if key == "" {
			key = m.MessageUID
		}
		if _, ok := groups[key]; !ok {
			groups[key] = &turnGroup{}
			groupOrder = append(groupOrder, key)
		}
		g := groups[key]

		switch m.Role {
		case schema.User:
			g.userMsg = m
		case schema.Assistant:
			if len(sm.ToolCalls) > 0 {
				if g.firstTCMsgUID == "" {
					g.firstTCMsgUID = m.MessageUID
				}
				for _, tc := range sm.ToolCalls {
					g.toolCalls = append(g.toolCalls, agui.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
						Extra:     guidanceMap[tc.ID],
					})
				}
			}
			// Capture Content independently — a row may carry both ToolCalls and Content
			// (written by savePartialAssistantMessage at the end of a tool-call chain).
			if sm.Content != "" {
				g.finalMsg = m
			}
		case agentmodel.RoleError:
			g.errorMsg = m
		}
	}

	result := make([]agui.Message, 0, len(msgs))
	for _, key := range groupOrder {
		g := groups[key]
		if g.userMsg != nil {
			var sm schema.Message
			if err := json.Unmarshal(g.userMsg.RawMessage, &sm); err == nil {
				result = append(result, agui.Message{
					ID:        g.userMsg.MessageUID,
					Role:      sm.Role,
					Content:   sm.Content,
					Status:    agui.StatusComplete,
					CreatedAt: g.userMsg.CreatedAt,
				})
			}
		}
		if g.finalMsg != nil || len(g.toolCalls) > 0 {
			out := agui.Message{
				Role:      schema.Assistant,
				Status:    agui.StatusComplete,
				ToolCalls: g.toolCalls,
			}
			if g.finalMsg != nil {
				var sm schema.Message
				if err := json.Unmarshal(g.finalMsg.RawMessage, &sm); err == nil {
					out.ID = g.finalMsg.MessageUID
					out.Content = sm.Content
					out.CreatedAt = g.finalMsg.CreatedAt
				}
			} else {
				// Turn interrupted before final row; use turnUID as synthetic ID.
				out.ID = key
			}
			result = append(result, out)
		}
		if g.errorMsg != nil {
			var sm schema.Message
			if err := json.Unmarshal(g.errorMsg.RawMessage, &sm); err == nil {
				result = append(result, agui.Message{
					ID:        g.errorMsg.MessageUID,
					Role:      sm.Role,
					Content:   sm.Content,
					Status:    agui.StatusError,
					CreatedAt: g.errorMsg.CreatedAt,
				})
			}
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
