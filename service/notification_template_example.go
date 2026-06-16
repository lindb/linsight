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
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// NotificationTemplateExampleService manages reusable sample alert payloads for template preview.
type NotificationTemplateExampleService interface {
	// CreateNotificationTemplateExample creates a new example.
	// If ex.IsDefault is true, any existing default for the same org is cleared atomically.
	CreateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) (string, error)
	// UpdateNotificationTemplateExample updates an existing example.
	// If ex.IsDefault is true, any existing default for the same org is cleared atomically.
	UpdateNotificationTemplateExample(ctx context.Context, ex *model.NotificationTemplateExample) error
	// DeleteNotificationTemplateExampleByUID removes an example.
	DeleteNotificationTemplateExampleByUID(ctx context.Context, uid string) error
	// GetNotificationTemplateExampleByUID returns an example by UID.
	GetNotificationTemplateExampleByUID(ctx context.Context, uid string) (*model.NotificationTemplateExample, error)
	// GetDefaultNotificationTemplateExample returns the default example for the current org, or
	// (nil, nil) when no default has been designated.
	GetDefaultNotificationTemplateExample(ctx context.Context) (*model.NotificationTemplateExample, error)
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

	err := srv.db.Transaction(func(tx dbpkg.DB) error {
		// Ensure at most one default per org — clear the current default before setting a new one.
		if ex.IsDefault {
			if err := tx.UpdateSingle(&model.NotificationTemplateExample{}, "is_default", false, "org_id=?", ex.OrgID); err != nil {
				return err
			}
		}
		return tx.Create(ex)
	})
	if err != nil {
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
	orgID := user.Org.ID

	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// Ensure at most one default per org when this example is being promoted to default.
		if ex.IsDefault {
			if err := tx.UpdateSingle(&model.NotificationTemplateExample{}, "is_default", false, "org_id=?", orgID); err != nil {
				return err
			}
		}
		fromDB.Name = ex.Name
		fromDB.Payload = ex.Payload
		fromDB.IsDefault = ex.IsDefault
		fromDB.UpdatedBy = user.User.ID
		return tx.Updates(fromDB, map[string]any{
			"name":       fromDB.Name,
			"payload":    fromDB.Payload,
			"is_default": fromDB.IsDefault,
			"updated_by": fromDB.UpdatedBy,
		}, "uid=? and org_id=?", ex.UID, orgID)
	})
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

func (srv *notificationTemplateExampleService) GetDefaultNotificationTemplateExample(ctx context.Context) (*model.NotificationTemplateExample, error) {
	signedUser := util.GetUser(ctx)
	var rs *model.NotificationTemplateExample
	err := srv.db.Get(&rs, "org_id=? and is_default=?", signedUser.Org.ID, true)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No default configured — callers treat nil as "not set".
		return nil, nil
	}
	if err != nil {
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
