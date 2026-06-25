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

	agentmodel "github.com/lindb/linsight/agent/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ── Conversations ─────────────────────────────────────────────────────────────

// GetOrCreateConversation returns the conversation with the given UID, or creates
// a new one if uid is empty. Returns the conversation and whether it was just created.
func (s *agentSettingsService) GetOrCreateConversation(
	ctx context.Context, uid string,
) (*agentmodel.AgentConversation, bool, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	if uid != "" {
		var conv agentmodel.AgentConversation
		if err := s.db.Get(&conv, "org_id = ? and uid = ?", orgID, uid); err != nil {
			return nil, false, err
		}
		return &conv, false, nil
	}

	conv := &agentmodel.AgentConversation{
		UID:    uuid.GenerateShortUUID(),
		OrgID:  orgID,
		UserID: user.User.ID,
	}
	conv.CreatedBy = user.User.ID
	conv.UpdatedBy = user.User.ID
	if err := s.db.Create(conv); err != nil {
		return nil, false, err
	}
	return conv, true, nil
}

// ListConversations returns paginated conversations for the user in ctx, newest first.
func (s *agentSettingsService) ListConversations(
	ctx context.Context, offset, limit int,
) ([]*agentmodel.AgentConversation, int64, error) {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	userID := user.User.ID
	total, err := s.db.Count(&agentmodel.AgentConversation{}, "org_id = ? and user_id = ?", orgID, userID)
	if err != nil {
		return nil, 0, err
	}
	var convs []*agentmodel.AgentConversation
	if err := s.db.FindForPaging(&convs, offset, limit, "id desc",
		"org_id = ? and user_id = ?", orgID, userID); err != nil {
		return nil, 0, err
	}
	return convs, total, nil
}

// DeleteConversation removes a conversation and all its messages in a transaction.
func (s *agentSettingsService) DeleteConversation(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	orgID := user.Org.ID
	return s.db.Transaction(func(tx dbpkg.DB) error {
		// Resolve the conversation so we can delete messages by conversation_id.
		var conv agentmodel.AgentConversation
		if err := tx.Get(&conv, "org_id = ? and uid = ?", orgID, uid); err != nil {
			return err
		}
		// Delete all messages in this conversation first (no FK constraint assumed).
		if err := tx.Delete(&agentmodel.AgentMessage{}, "conversation_id = ?", conv.ID); err != nil {
			return err
		}
		return tx.Delete(&agentmodel.AgentConversation{}, "id = ?", conv.ID)
	})
}

// ── Messages ──────────────────────────────────────────────────────────────────

// GetRecentMessages returns up to limit messages for the conversation, ordered
// by ID descending (newest first), then reversed so the oldest is at index 0.
func (s *agentSettingsService) GetRecentMessages(ctx context.Context, conversationID int64, limit int) ([]*agentmodel.AgentMessage, error) {
	var msgs []*agentmodel.AgentMessage
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
func (s *agentSettingsService) SaveMessage(ctx context.Context, msg *agentmodel.AgentMessage) error {
	return s.db.Create(msg)
}
