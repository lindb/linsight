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
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

// ContactPointStatusService persists and retrieves per-channel delivery outcomes.
type ContactPointStatusService interface {
	// UpsertChannelStatus inserts or updates the delivery status for one channel of a contact point.
	// Keyed by (orgID, contactPointUID, channelUID).
	UpsertChannelStatus(orgID int64, cpUID, channelUID, channelType, status, errMsg string, sentAt, durationMs int64) error
	// BatchGetByContactPoints returns the latest delivery statuses for multiple contact points,
	// grouped by contact-point UID. One DB round-trip via IN clause.
	BatchGetByContactPoints(orgID int64, cpUIDs []string) (map[string][]model.ContactPointStatus, error)
}

type contactPointStatusService struct {
	db dbpkg.DB
}

// NewContactPointStatusService creates a ContactPointStatusService backed by db.
func NewContactPointStatusService(db dbpkg.DB) ContactPointStatusService {
	return &contactPointStatusService{db: db}
}

func (srv *contactPointStatusService) UpsertChannelStatus(
	orgID int64, cpUID, channelUID, channelType, status, errMsg string, sentAt, durationMs int64,
) error {
	// Check whether a row already exists for this (org, contact-point, channel-uid) triple.
	var existing model.ContactPointStatus
	err := srv.db.Get(&existing,
		"org_id=? and contact_point_uid=? and channel_uid=?",
		orgID, cpUID, channelUID)

	if err != nil {
		// No existing row — insert a new one.
		return srv.db.Create(&model.ContactPointStatus{
			OrgID:           orgID,
			ContactPointUID: cpUID,
			ChannelUID:      channelUID,
			ChannelType:     channelType,
			Status:          status,
			ErrorMessage:    errMsg,
			LastSentAt:      sentAt,
			DurationMs:      durationMs,
		})
	}

	// Row exists — update only the mutable columns so we don't touch the unique-key columns.
	return srv.db.Updates(&existing,
		map[string]any{
			"channel_type":  channelType,
			"status":        status,
			"error_message": errMsg,
			"last_sent_at":  sentAt,
			"duration_ms":   durationMs,
		},
		"id=?", existing.ID)
}

func (srv *contactPointStatusService) BatchGetByContactPoints(
	orgID int64, cpUIDs []string,
) (map[string][]model.ContactPointStatus, error) {
	if len(cpUIDs) == 0 {
		return nil, nil
	}
	var rows []model.ContactPointStatus
	if err := srv.db.Find(&rows, "org_id=? and contact_point_uid in ?", orgID, cpUIDs); err != nil {
		return nil, err
	}
	result := make(map[string][]model.ContactPointStatus, len(cpUIDs))
	for _, r := range rows {
		result[r.ContactPointUID] = append(result[r.ContactPointUID], r)
	}
	return result, nil
}
