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

	"gorm.io/gorm"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

// NotificationService manages global notification settings (per-plugin credentials such as SMTP / Slack token).
// The legacy Notification CRUD (named receiver lists) has been removed; use ContactPoint instead.
type NotificationService interface {
	GetAllNotificationSettings(ctx context.Context) (rs []model.NotificationSetting, err error)
	SaveNotificationSetting(ctx context.Context, setting *model.NotificationSetting) error
	DeleteNotificationSetting(ctx context.Context, notificationType string) error
	GetNotificationSettingsByOrg(orgID int64) (rs []model.NotificationSetting, err error)
}

type notificationService struct {
	db dbpkg.DB
}

func NewNotificationService(db dbpkg.DB) NotificationService {
	return &notificationService{
		db: db,
	}
}

func (srv *notificationService) GetAllNotificationSettings(ctx context.Context) (rs []model.NotificationSetting, err error) {
	user := util.GetUser(ctx)
	return srv.GetNotificationSettingsByOrg(user.Org.ID)
}

func (srv *notificationService) GetNotificationSettingsByOrg(orgID int64) (rs []model.NotificationSetting, err error) {
	if err := srv.db.Find(&rs, "org_id=?", orgID); err != nil {
		return nil, err
	}
	return
}

func (srv *notificationService) SaveNotificationSetting(ctx context.Context, setting *model.NotificationSetting) error {
	user := util.GetUser(ctx)
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		var settingFromDB model.NotificationSetting
		err := tx.Get(&settingFromDB, "org_id=? and type=?", user.Org.ID, setting.Type)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting.OrgID = user.Org.ID
			return tx.Create(&setting)
		}
		if err != nil {
			return err
		}
		settingFromDB.Config = setting.Config
		return tx.Update(settingFromDB, "org_id=? and type=?", user.Org.ID, setting.Type)
	})
}

func (srv *notificationService) DeleteNotificationSetting(ctx context.Context, notificationType string) error {
	user := util.GetUser(ctx)
	return srv.db.Delete(&model.NotificationSetting{}, "org_id=? and type=?", user.Org.ID, notificationType)
}
