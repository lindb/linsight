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
	"errors"
	"fmt"

	"gorm.io/gorm"

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

// ErrMessageNotInConversation is returned when the messageUID does not belong
// to the given conversationUID.
var ErrMessageNotInConversation = fmt.Errorf("message does not belong to this conversation")

// ErrInvalidFeedbackReason is returned when an unknown reason tag is submitted.
var ErrInvalidFeedbackReason = fmt.Errorf("invalid feedback reason tag")

// FeedbackService manages user feedback on assistant messages.
type FeedbackService interface {
	// UpsertFeedback creates or updates the feedback for a single message.
	// Business rules enforced:
	//   - Caller must be the owner of the conversation identified by convUID.
	//   - messageUID must belong to convUID (prevents cross-conversation injection).
	//   - Each element of req.Reasons must be a member of ValidFeedbackReasons.
	// Returns the full MessageFeedback record after the upsert.
	UpsertFeedback(ctx context.Context, convUID, messageUID string, req *agentapi.FeedbackRequest) (*agentmodel.MessageFeedback, error)

	// GetFeedback returns the feedback for a specific message, scoped to the caller (org+user).
	// Returns gorm.ErrRecordNotFound if no feedback has been submitted yet.
	GetFeedback(ctx context.Context, convUID, messageUID string) (*agentmodel.MessageFeedback, error)
}

type feedbackService struct {
	db      dbpkg.DB
	convSrv ConversationService
}

// NewFeedbackService creates a FeedbackService backed by db.
// convSrv is injected so that ownership and org-isolation checks reuse
// GetConversation, which scopes by org+user from context.
func NewFeedbackService(db dbpkg.DB, convSrv ConversationService) FeedbackService {
	return &feedbackService{db: db, convSrv: convSrv}
}

// UpsertFeedback creates or replaces the feedback for one message.
func (s *feedbackService) UpsertFeedback(
	ctx context.Context, convUID, messageUID string, req *agentapi.FeedbackRequest,
) (*agentmodel.MessageFeedback, error) {
	user := util.GetUser(ctx)

	// Step 1: Load conversation — scoped to org+user via GetConversation.
	conv, err := s.convSrv.GetConversation(ctx, convUID)
	if err != nil {
		return nil, err
	}

	// Step 2: Verify the messageUID belongs to this conversation.
	// Uses the physical (conversation_uid, message_uid) indexed columns — no join.
	var msg agentmodel.Message
	if err := s.db.Get(&msg, "message_uid = ? and conversation_uid = ?", messageUID, convUID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotInConversation
		}
		return nil, err
	}

	// Step 3: Validate reason tags before any write.
	for _, r := range req.Reasons {
		if _, ok := agentmodel.ValidFeedbackReasons[agentmodel.FeedbackReason(r)]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrInvalidFeedbackReason, r)
		}
	}

	// Step 4: Marshal reasons to JSON (nil slice → omit column).
	var reasonsJSON json.RawMessage
	if len(req.Reasons) > 0 {
		reasonsJSON, err = json.Marshal(req.Reasons)
		if err != nil {
			return nil, fmt.Errorf("marshal reasons: %w", err)
		}
	}

	// Step 5: Manual upsert — db.Get to detect existence, then Create or Update.
	// This pattern is consistent with the rest of the project (no OnConflict clauses).
	var existing agentmodel.MessageFeedback
	dbErr := s.db.Get(&existing, "message_uid = ?", messageUID)
	if dbErr != nil {
		if !errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return nil, dbErr
		}
		// No existing row — create a new feedback record.
		fb := &agentmodel.MessageFeedback{
			ConversationID:   conv.ID,
			ConversationUID:  convUID,
			MessageUID:       messageUID,
			UserID:           user.User.ID,
			Rating:           agentmodel.FeedbackRating(req.Rating),
			Reasons:          reasonsJSON,
			Comment:          req.Comment,
			CorrectedContent: req.CorrectedContent,
		}
		fb.CreatedBy = user.User.ID
		fb.UpdatedBy = user.User.ID
		if err := s.db.Create(fb); err != nil {
			return nil, err
		}
		return fb, nil
	}

	// Existing row found — update only the mutable feedback fields.
	updates := map[string]any{
		"rating":            agentmodel.FeedbackRating(req.Rating),
		"reasons":           reasonsJSON,
		"comment":           req.Comment,
		"corrected_content": req.CorrectedContent,
		"updated_by":        user.User.ID,
	}
	if err := s.db.RawDB().Model(&agentmodel.MessageFeedback{}).
		Where("message_uid = ? and user_id = ?", messageUID, user.User.ID).
		Updates(updates).Error; err != nil {
		return nil, err
	}
	// Return refreshed state without an extra round-trip.
	existing.Rating = agentmodel.FeedbackRating(req.Rating)
	existing.Reasons = reasonsJSON
	existing.Comment = req.Comment
	existing.CorrectedContent = req.CorrectedContent
	existing.UpdatedBy = user.User.ID
	return &existing, nil
}

// GetFeedback returns the feedback for a specific message, scoped to the caller's org.
func (s *feedbackService) GetFeedback(
	ctx context.Context, convUID, messageUID string,
) (*agentmodel.MessageFeedback, error) {
	// Verify the conversation exists and belongs to the caller (org+user scoped).
	if _, err := s.convSrv.GetConversation(ctx, convUID); err != nil {
		return nil, err
	}

	var fb agentmodel.MessageFeedback
	if err := s.db.Get(&fb, "message_uid = ? and conversation_uid = ?", messageUID, convUID); err != nil {
		return nil, err
	}
	return &fb, nil
}
