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

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// AlertSilenceService manages alert silences.
type AlertSilenceService interface {
	// CreateAlertSilence creates a new silence.
	CreateAlertSilence(ctx context.Context, silence *model.AlertSilence) (string, error)
	// UpdateAlertSilence updates an existing silence (e.g. to extend or shorten duration).
	UpdateAlertSilence(ctx context.Context, silence *model.AlertSilence) error
	// DeleteAlertSilenceByUID immediately expires a silence.
	DeleteAlertSilenceByUID(ctx context.Context, uid string) error
	// GetAlertSilenceByUID returns a silence by UID.
	GetAlertSilenceByUID(ctx context.Context, uid string) (*model.AlertSilence, error)
	// SearchAlertSilences queries silences with optional filters.
	SearchAlertSilences(ctx context.Context, req *model.SearchAlertSilenceRequest) (rs []model.AlertSilence, total int64, err error)
	// GetActiveSilencesByOrg returns all currently active silences for an org (used by the alert engine).
	GetActiveSilencesByOrg(orgID int64, now int64) ([]*model.AlertSilence, error)
}

type alertSilenceService struct {
	db dbpkg.DB
}

// NewAlertSilenceService creates an AlertSilenceService instance.
func NewAlertSilenceService(db dbpkg.DB) AlertSilenceService {
	return &alertSilenceService{db: db}
}

func (srv *alertSilenceService) CreateAlertSilence(ctx context.Context, silence *model.AlertSilence) (string, error) {
	silence.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	silence.OrgID = user.Org.ID
	silence.CreatedBy = user.User.ID
	silence.UpdatedBy = user.User.ID
	if silence.CreatedByName == "" {
		silence.CreatedByName = user.User.Name
	}
	if err := srv.db.Create(silence); err != nil {
		return "", err
	}
	return silence.UID, nil
}

func (srv *alertSilenceService) UpdateAlertSilence(ctx context.Context, silence *model.AlertSilence) error {
	fromDB, err := srv.GetAlertSilenceByUID(ctx, silence.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Matchers = silence.Matchers
	fromDB.StartsAt = silence.StartsAt
	fromDB.EndsAt = silence.EndsAt
	fromDB.Comment = silence.Comment
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", silence.UID, user.Org.ID)
}

func (srv *alertSilenceService) DeleteAlertSilenceByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.AlertSilence{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *alertSilenceService) GetAlertSilenceByUID(ctx context.Context, uid string) (rs *model.AlertSilence, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertSilenceService) SearchAlertSilences(ctx context.Context, req *model.SearchAlertSilenceRequest) (rs []model.AlertSilence, total int64, err error) {
	signedUser := util.GetUser(ctx)
	where := "org_id=?"
	params := []any{signedUser.Org.ID}

	count, err := srv.db.Count(&model.AlertSilence{}, where, params...)
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

func (srv *alertSilenceService) GetActiveSilencesByOrg(orgID int64, now int64) ([]*model.AlertSilence, error) {
	var rs []*model.AlertSilence
	// active silences: starts_at <= now < ends_at
	if err := srv.db.Find(&rs, "org_id=? and starts_at<=? and ends_at>?", orgID, now, now); err != nil {
		return nil, err
	}
	return rs, nil
}
