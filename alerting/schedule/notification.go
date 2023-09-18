package schedule

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

type NotificationManager interface {
	LoadNotificationSettings()
	GetNotificationSetting(notifiyType string) (*model.NotificationSetting, bool)

	Notifiy(rule *AlertRule, alerts []*model.Alert) error
}

type notificationManager struct {
	orgID                int64
	deps                 *Deps
	notificationSettings map[string]*model.NotificationSetting

	lock sync.RWMutex

	logger logger.Logger
}

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
		mgr.logger.Error("load notification settings failure", logger.Any("org", mgr.orgID), logger.Error(err))
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

func (mgr *notificationManager) GetNotificationSetting(notifiyType string) (*model.NotificationSetting, bool) {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()

	s, ok := mgr.notificationSettings[notifiyType]
	return s, ok
}

func (mgr *notificationManager) Notifiy(rule *AlertRule, alerts []*model.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	notifications := rule.GetNotifications()
	for _, nUID := range notifications {
		notification, err := mgr.deps.NotificationSrv.GetNotificationByOrgAndUID(mgr.orgID, nUID)
		if err != nil {
			return err
		}
		receivers, err := notification.ToReceviers()
		if err != nil {
			return err
		}
		for _, receiver := range receivers {
			n, ok := mgr.GetNotificationSetting(receiver.Type)
			if !ok {
				mgr.logger.Warn("notification setting not found", logger.Any("org", mgr.orgID), logger.Any("type", receiver.Type))
				continue
			}

			d, _ := n.Config.MarshalJSON()
			slack, _ := types.GetReceiverPlugin(receiver.Type)
			cli, err := slack.NewFn(json.RawMessage(d), receiver.Config)
			if err != nil {
				return err
			}
			cli.Notify(context.TODO(), alerts)
		}
	}

	return nil
}
