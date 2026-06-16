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

// NotificationTemplateService manages reusable notification templates.
type NotificationTemplateService interface {
	// CreateNotificationTemplate creates a new template.
	CreateNotificationTemplate(ctx context.Context, tmpl *model.NotificationTemplate) (string, error)
	// UpdateNotificationTemplate updates an existing template.
	UpdateNotificationTemplate(ctx context.Context, tmpl *model.NotificationTemplate) error
	// DeleteNotificationTemplateByUID removes a template.
	DeleteNotificationTemplateByUID(ctx context.Context, uid string) error
	// GetNotificationTemplateByUID returns a template by UID.
	GetNotificationTemplateByUID(ctx context.Context, uid string) (*model.NotificationTemplate, error)
	// SearchNotificationTemplates queries templates with optional filters.
	SearchNotificationTemplates(ctx context.Context, req *model.SearchNotificationTemplateRequest) (rs []model.NotificationTemplate, total int64, err error)
	// GetAllNotificationTemplates returns all templates for the current user's org (used by HTTP handlers).
	GetAllNotificationTemplates(ctx context.Context) ([]model.NotificationTemplate, error)
	// GetTemplateByOrgAndUID returns a template without a signed-in user context (used by the alert engine).
	GetTemplateByOrgAndUID(orgID int64, uid string) (*model.NotificationTemplate, error)
	// GetAllTemplatesByOrgID returns all templates for an org without a signed-in user context (used by the alert engine).
	GetAllTemplatesByOrgID(orgID int64) ([]model.NotificationTemplate, error)
}

type notificationTemplateService struct {
	db dbpkg.DB
}

// NewNotificationTemplateService creates a NotificationTemplateService instance.
func NewNotificationTemplateService(db dbpkg.DB) NotificationTemplateService {
	return &notificationTemplateService{db: db}
}

func (srv *notificationTemplateService) CreateNotificationTemplate(ctx context.Context, tmpl *model.NotificationTemplate) (string, error) {
	tmpl.UID = uuid.GenerateShortUUID()
	user := util.GetUser(ctx)
	tmpl.OrgID = user.Org.ID
	tmpl.CreatedBy = user.User.ID
	tmpl.UpdatedBy = user.User.ID
	if err := srv.db.Create(tmpl); err != nil {
		return "", err
	}
	return tmpl.UID, nil
}

func (srv *notificationTemplateService) UpdateNotificationTemplate(ctx context.Context, tmpl *model.NotificationTemplate) error {
	fromDB, err := srv.GetNotificationTemplateByUID(ctx, tmpl.UID)
	if err != nil {
		return err
	}
	user := util.GetUser(ctx)
	fromDB.Name = tmpl.Name
	fromDB.Template = tmpl.Template
	fromDB.UpdatedBy = user.User.ID
	return srv.db.Update(fromDB, "uid=? and org_id=?", tmpl.UID, user.Org.ID)
}

func (srv *notificationTemplateService) DeleteNotificationTemplateByUID(ctx context.Context, uid string) error {
	signedUser := util.GetUser(ctx)
	return srv.db.Delete(&model.NotificationTemplate{}, "uid=? and org_id=?", uid, signedUser.Org.ID)
}

func (srv *notificationTemplateService) GetNotificationTemplateByUID(ctx context.Context, uid string) (rs *model.NotificationTemplate, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *notificationTemplateService) SearchNotificationTemplates(ctx context.Context, req *model.SearchNotificationTemplateRequest) (rs []model.NotificationTemplate, total int64, err error) {
	signedUser := util.GetUser(ctx)
	conditions := []string{"org_id=?"}
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.NotificationTemplate{}, where, params...)
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

func (srv *notificationTemplateService) GetTemplateByOrgAndUID(orgID int64, uid string) (rs *model.NotificationTemplate, err error) {
	if err := srv.db.Get(&rs, "uid=? and org_id=?", uid, orgID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *notificationTemplateService) GetAllNotificationTemplates(ctx context.Context) (rs []model.NotificationTemplate, err error) {
	signedUser := util.GetUser(ctx)
	if err := srv.db.Find(&rs, "org_id=?", signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}

func (srv *notificationTemplateService) GetAllTemplatesByOrgID(orgID int64) (rs []model.NotificationTemplate, err error) {
	if err := srv.db.Find(&rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return rs, nil
}
