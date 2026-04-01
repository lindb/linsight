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

// AlertReceiverService manages alert receiver configurations.
type AlertReceiverService interface {
	// CreateAlertReceiver creates a new receiver.
	CreateAlertReceiver(ctx context.Context, receiver *model.AlertReceiver) (string, error)
	// UpdateAlertReceiver updates an existing receiver.
	UpdateAlertReceiver(ctx context.Context, receiver *model.AlertReceiver) error
	// DeleteAlertReceiverByUID removes a receiver.
	DeleteAlertReceiverByUID(ctx context.Context, uid string) error
	// GetAlertReceiverByUID returns a receiver by UID.
	GetAlertReceiverByUID(ctx context.Context, uid string) (*model.AlertReceiver, error)
	// SearchAlertReceivers queries receivers with optional filters.
	SearchAlertReceivers(ctx context.Context, req *model.SearchAlertReceiverRequest) (rs []model.AlertReceiver, total int64, err error)
	// GetAlertReceiverByOrgAndUID returns a receiver accessible by the alert engine without a signed-in user context.
	GetAlertReceiverByOrgAndUID(orgID int64, uid string) (*model.AlertReceiver, error)
}

type alertReceiverService struct {
	db dbpkg.DB
}

// NewAlertReceiverService creates an AlertReceiverService instance.
func NewAlertReceiverService(db dbpkg.DB) AlertReceiverService {
	return &alertReceiverService{db: db}
}

func (srv *alertReceiverService) CreateAlertReceiver(ctx context.Context, receiver *model.AlertReceiver) (string, error) {
	receiver.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	receiver.OrgID = user.Org.ID
	receiver.CreatedBy = user.User.ID
	receiver.UpdatedBy = user.User.ID
	if err := srv.db.Create(receiver); err != nil {
		return "", err
	}
	return receiver.UID, nil
}

func (srv *alertReceiverService) UpdateAlertReceiver(ctx context.Context, receiver *model.AlertReceiver) error {
	fromDB, err := srv.GetAlertReceiverByUID(ctx, receiver.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = receiver.Name
	fromDB.Channels = receiver.Channels
	fromDB.TemplateUID = receiver.TemplateUID
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", receiver.UID, user.Org.ID)
}

func (srv *alertReceiverService) DeleteAlertReceiverByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.AlertReceiver{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *alertReceiverService) GetAlertReceiverByUID(ctx context.Context, uid string) (rs *model.AlertReceiver, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *alertReceiverService) SearchAlertReceivers(ctx context.Context, req *model.SearchAlertReceiverRequest) (rs []model.AlertReceiver, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.AlertReceiver{}, where, params...)
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

func (srv *alertReceiverService) GetAlertReceiverByOrgAndUID(orgID int64, uid string) (rs *model.AlertReceiver, err error) {
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
