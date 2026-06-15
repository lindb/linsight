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
	"strings"

	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// ContactPointService manages contact point configurations.
type ContactPointService interface {
	// CreateContactPoint creates a new contact point.
	CreateContactPoint(ctx context.Context, cp *model.ContactPoint) (string, error)
	// UpdateContactPoint updates an existing contact point.
	UpdateContactPoint(ctx context.Context, cp *model.ContactPoint) error
	// DeleteContactPointByUID removes a contact point.
	DeleteContactPointByUID(ctx context.Context, uid string) error
	// GetContactPointByUID returns a contact point by UID.
	GetContactPointByUID(ctx context.Context, uid string) (*model.ContactPoint, error)
	// SearchContactPoints queries contact points with optional filters.
	SearchContactPoints(ctx context.Context, req *model.SearchContactPointRequest) (rs []model.ContactPoint, total int64, err error)
	// GetContactPointByOrgAndUID returns a contact point accessible by the alert engine without a signed-in user context.
	GetContactPointByOrgAndUID(orgID int64, uid string) (*model.ContactPoint, error)
}

type contactPointService struct {
	db dbpkg.DB
}

// NewContactPointService creates a ContactPointService instance.
func NewContactPointService(db dbpkg.DB) ContactPointService {
	return &contactPointService{db: db}
}

// ensureChannelUIDs decodes the channels JSON, assigns a short UUID to any channel
// that lacks one, and re-encodes the result back onto the ContactPoint.
// This guarantees each channel has a stable identifier for ContactPointStatus keying.
func ensureChannelUIDs(cp *model.ContactPoint) error {
	channels, err := cp.ToChannels()
	if err != nil || len(channels) == 0 {
		return err
	}
	changed := false
	for i := range channels {
		if channels[i].UID == "" {
			channels[i].UID = uuid.GenerateShortUUID()
			changed = true
		}
	}
	if !changed {
		return nil
	}
	data, err := json.Marshal(channels)
	if err != nil {
		return err
	}
	cp.Channels = datatypes.JSON(data)
	return nil
}

func (srv *contactPointService) CreateContactPoint(ctx context.Context, cp *model.ContactPoint) (string, error) {
	cp.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	cp.OrgID = user.Org.ID
	cp.CreatedBy = user.User.ID
	cp.UpdatedBy = user.User.ID
	// assign UIDs to channels that don't have one before persisting
	if err := ensureChannelUIDs(cp); err != nil {
		return "", err
	}
	if err := srv.db.Create(cp); err != nil {
		return "", err
	}
	return cp.UID, nil
}

func (srv *contactPointService) UpdateContactPoint(ctx context.Context, cp *model.ContactPoint) error {
	fromDB, err := srv.GetContactPointByUID(ctx, cp.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = cp.Name
	fromDB.Channels = cp.Channels
	fromDB.TemplateUID = cp.TemplateUID
	fromDB.UpdatedBy = user.User.ID
	// fill missing UIDs before persisting so new channels always get a stable identifier
	if err := ensureChannelUIDs(fromDB); err != nil {
		return err
	}
	return srv.db.Update(fromDB, "uid=? and org_id=?", cp.UID, user.Org.ID)
}

func (srv *contactPointService) DeleteContactPointByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.ContactPoint{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *contactPointService) GetContactPointByUID(ctx context.Context, uid string) (rs *model.ContactPoint, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *contactPointService) SearchContactPoints(ctx context.Context, req *model.SearchContactPointRequest) (rs []model.ContactPoint, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.ContactPoint{}, where, params...)
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

func (srv *contactPointService) GetContactPointByOrgAndUID(orgID int64, uid string) (rs *model.ContactPoint, err error) {
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
