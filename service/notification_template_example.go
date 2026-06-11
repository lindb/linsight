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

// NotificationTemplateExampleService manages reusable sample alert payloads for template preview.
type NotificationTemplateExampleService interface {
	// CreateNotificationTemplateExample creates a new example.
	CreateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) (string, error)
	// UpdateNotificationTemplateExample updates an existing example.
	UpdateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) error
	// DeleteNotificationTemplateExampleByUID removes an example.
	DeleteNotificationTemplateExampleByUID(ctx context.Context, uid string) error
	// GetNotificationTemplateExampleByUID returns an example by UID.
	GetNotificationTemplateExampleByUID(ctx context.Context, uid string) (*model.NotificationTemplateExample, error)
	// SearchNotificationTemplateExamples queries examples with optional filters.
	SearchNotificationTemplateExamples(ctx context.Context, req *model.SearchNotificationTemplateExampleRequest) (rs []model.NotificationTemplateExample, total int64, err error)
}

type notificationTemplateExampleService struct {
	db dbpkg.DB
}

// NewNotificationTemplateExampleService creates a NotificationTemplateExampleService instance.
func NewNotificationTemplateExampleService(db dbpkg.DB) NotificationTemplateExampleService {
	return &notificationTemplateExampleService{db: db}
}

func (srv *notificationTemplateExampleService) CreateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) (string, error) {
	ex.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	ex.OrgID = user.Org.ID
	ex.CreatedBy = user.User.ID
	ex.UpdatedBy = user.User.ID
	if err := srv.db.Create(ex); err != nil {
		return "", err
	}
	return ex.UID, nil
}

func (srv *notificationTemplateExampleService) UpdateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) error {
	fromDB, err := srv.GetNotificationTemplateExampleByUID(ctx, ex.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = ex.Name
	fromDB.Payload = ex.Payload
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", ex.UID, user.Org.ID)
}

func (srv *notificationTemplateExampleService) DeleteNotificationTemplateExampleByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.NotificationTemplateExample{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *notificationTemplateExampleService) GetNotificationTemplateExampleByUID(ctx context.Context, uid string) (rs *model.NotificationTemplateExample, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *notificationTemplateExampleService) SearchNotificationTemplateExamples(ctx context.Context, req *model.SearchNotificationTemplateExampleRequest) (rs []model.NotificationTemplateExample, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.NotificationTemplateExample{}, where, params...)
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
