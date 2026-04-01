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
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// AlertEventService manages the persistence and retrieval of alert events.
type AlertEventService interface {
	// SearchAlertEvents queries alert events with optional filters.
	SearchAlertEvents(ctx context.Context, req *model.SearchAlertEventRequest) (rs []model.AlertEvent, total int64, err error)
	// GetAlertEventByUID returns a single alert event by UID.
	GetAlertEventByUID(ctx context.Context, uid string) (*model.AlertEvent, error)
	// UpsertAlertEvent creates or updates an alert event identified by its fingerprint.
	UpsertAlertEvent(orgID int64, event *model.AlertEvent) error
	// GetActiveAlertEvents returns all non-resolved events for an org.
	GetActiveAlertEvents(orgID int64) ([]*model.AlertEvent, error)
}

type alertEventService struct {
	db dbpkg.DB
}

// NewAlertEventService creates an AlertEventService instance.
func NewAlertEventService(db dbpkg.DB) AlertEventService {
	return &alertEventService{db: db}
}

func (srv *alertEventService) SearchAlertEvents(ctx context.Context, req *model.SearchAlertEventRequest) (rs []model.AlertEvent, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}

	if req.RuleUID != "" {
		conditions = append(conditions, "rule_uid=?")
		params = append(params, req.RuleUID)
	}
	if req.State > 0 {
		conditions = append(conditions, "state=?")
		params = append(params, req.State)
	}

	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertEvent{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

func (srv *alertEventService) GetAlertEventByUID(ctx context.Context, uid string) (rs *model.AlertEvent, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertEventService) UpsertAlertEvent(orgID int64, event *model.AlertEvent) error {
	var existing model.AlertEvent
	err := srv.db.Get(&existing, "fingerprint=? and org_id=?", event.Fingerprint, orgID)
	if err != nil {
		// record not found — create new
		if event.UID == "" {
			event.UID = uuid.GenerateShortUUID()
		}
		event.OrgID = orgID
		return srv.db.Create(event)
	}
	// update existing record
	existing.State = event.State
	existing.Value = event.Value
	existing.EndsAt = event.EndsAt
	existing.NotifiedAt = event.NotifiedAt
	existing.Annotations = event.Annotations
	if event.StartsAt > 0 && existing.StartsAt == 0 {
		existing.StartsAt = event.StartsAt
	}
	if event.PendingSince > 0 && existing.PendingSince == 0 {
		existing.PendingSince = event.PendingSince
	}
	return srv.db.Update(&existing, "fingerprint=? and org_id=?", event.Fingerprint, orgID)
}

func (srv *alertEventService) GetActiveAlertEvents(orgID int64) ([]*model.AlertEvent, error) {
	var rs []*model.AlertEvent
	if err := srv.db.Find(&rs, "org_id=? and state in (?,?)", orgID, model.AlertEventPending, model.AlertEventFiring); err != nil {
		return nil, err
	}
	return rs, nil
}
