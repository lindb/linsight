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

	basemodel "github.com/lindb/linsight/model"
)

// FeedbackRating is the polarity signal attached to a user's message feedback.
// 1 = thumbs up, -1 = thumbs down. 0 is the zero value and must not appear in
// persisted rows.
type FeedbackRating int8

const (
	FeedbackRatingThumbsUp   FeedbackRating = 1
	FeedbackRatingThumbsDown FeedbackRating = -1
)

// FeedbackReason is a predefined tag that categorises why the user rated a message.
// Tags are stored as a JSON array in the reasons column; unknown tags are rejected
// by the service layer before any write.
type FeedbackReason string

const (
	// Negative quality signals — typically paired with FeedbackRatingThumbsDown.
	FeedbackReasonToolCallError FeedbackReason = "tool_call_error"
	FeedbackReasonHallucination FeedbackReason = "hallucination"
	FeedbackReasonBadFormat     FeedbackReason = "bad_format"
	FeedbackReasonOutdatedInfo  FeedbackReason = "outdated_info"
	FeedbackReasonIncomplete    FeedbackReason = "incomplete"
	FeedbackReasonOffTopic      FeedbackReason = "off_topic"

	// Positive quality signals — typically paired with FeedbackRatingThumbsUp.
	FeedbackReasonLogicPerfect FeedbackReason = "logic_perfect"
	FeedbackReasonHelpful      FeedbackReason = "helpful"
	FeedbackReasonConcise      FeedbackReason = "concise"
)

// ValidFeedbackReasons is the complete set of accepted reason tags.
// The service layer rejects any tag not present in this map so that unknown
// strings from buggy or malicious clients never pollute the training data.
// Adding a new valid tag requires only a one-line change here — no migration.
var ValidFeedbackReasons = map[FeedbackReason]struct{}{
	FeedbackReasonToolCallError: {},
	FeedbackReasonHallucination: {},
	FeedbackReasonBadFormat:     {},
	FeedbackReasonOutdatedInfo:  {},
	FeedbackReasonIncomplete:    {},
	FeedbackReasonOffTopic:      {},
	FeedbackReasonLogicPerfect:  {},
	FeedbackReasonHelpful:       {},
	FeedbackReasonConcise:       {},
}

// MessageFeedback stores one user rating for one assistant message.
// Invariant: at most one row per MessageUID (enforced by uniqueIndex).
// Only the conversation owner (UserID == conversations.user_id) may write feedback.
// Any viewer in the same org may read it.
type MessageFeedback struct {
	basemodel.BaseModel

	// ConversationID is the numeric FK to conversations.id.
	// Indexed for cascade-delete efficiency (delete by conversation_id without a join).
	ConversationID int64 `json:"-" gorm:"column:conversation_id;index"`

	// ConversationUID is the public string key of the owning conversation.
	// Redundant physical column — mirrors conversations.uid — so ownership
	// checks use a single WHERE clause without an extra join.
	ConversationUID string `json:"conversationUid" gorm:"column:conversation_uid;index"`

	// MessageUID is the AG-UI message ID (mirrors messages.message_uid).
	// uniqueIndex enforces the one-feedback-per-message invariant at DB level.
	MessageUID string `json:"messageUid" gorm:"column:message_uid;uniqueIndex"`

	// UserID is the submitting user, denormalized for O(1) ownership checks.
	UserID int64 `json:"-" gorm:"column:user_id;index"`

	// Rating is +1 (thumbs up) or -1 (thumbs down). Required.
	Rating FeedbackRating `json:"rating" gorm:"column:rating"`

	// Reasons is a JSON array of predefined FeedbackReason strings.
	// Stored as native JSON (MySQL/PostgreSQL) or TEXT (SQLite).
	Reasons json.RawMessage `json:"reasons,omitempty" gorm:"column:reasons;type:json"`

	// Comment is free-text qualitative feedback from the user. Optional.
	Comment string `json:"comment,omitempty" gorm:"column:comment;type:text"`

	// CorrectedContent is the user's manual rewrite of the assistant message.
	// Used as gold-label annotation for SFT/DPO fine-tuning pipelines. Optional.
	CorrectedContent string `json:"correctedContent,omitempty" gorm:"column:corrected_content;type:text"`
}

// TableName specifies the GORM table name.
func (MessageFeedback) TableName() string { return "message_feedbacks" }
