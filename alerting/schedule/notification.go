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

package schedule

import (
	"sync"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// NotificationManager is kept for backward compatibility with the schedule bootstrapping code.
// Actual notification routing is now handled by alerting/manager.AlertManager.
// This manager only handles loading the global notification settings used by legacy notification plugins.
type NotificationManager interface {
	LoadNotificationSettings()
	GetNotificationSetting(notifyType string) (*model.NotificationSetting, bool)
}

type notificationManager struct {
	orgID                int64
	deps                 *Deps
	notificationSettings map[string]*model.NotificationSetting

	lock sync.RWMutex

	logger logger.Logger
}

// NewNotificationManager creates a NotificationManager instance.
func NewNotificationManager(orgID int64, deps *Deps) NotificationManager {
	return &notificationManager{
		orgID:                orgID,
		deps:                 deps,
		notificationSettings: make(map[string]*model.NotificationSetting),
		logger:               logger.GetLogger("Alerting", "NotificationManager"),
	}
}

func (mgr *notificationManager) LoadNotificationSettings() {
	notificationSettings, err := mgr.deps.NotificationSrv.GetNotificationSettingsByOrg(mgr.orgID)
	if err != nil {
		mgr.logger.Error("load notification settings failure",
			logger.Any("org", mgr.orgID),
			logger.Error(err))
		return
	}

	notificationSettingMap := make(map[string]*model.NotificationSetting)
	for idx := range notificationSettings {
		ns := notificationSettings[idx]
		notificationSettingMap[ns.Type] = &ns
	}

	mgr.lock.Lock()
	mgr.notificationSettings = notificationSettingMap
	mgr.lock.Unlock()
}

func (mgr *notificationManager) GetNotificationSetting(notifyType string) (*model.NotificationSetting, bool) {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	s, ok := mgr.notificationSettings[notifyType]
	return s, ok
}
