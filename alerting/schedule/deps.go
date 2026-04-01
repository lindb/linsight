package schedule

import (
	"github.com/lindb/linsight/alerting/manager"
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

// Deps holds all external dependencies required by the alert scheduling subsystem.
type Deps struct {
	Opts            *Options
	DatasourceSrv   service.DatasourceService
	AlertRuleSrv    service.AlertRuleService
	NotificationSrv service.NotificationService
	DatasourceMgr   datasource.Manager

	// AlertMgr receives alert events and handles routing, silencing and notification dispatch.
	AlertMgr manager.AlertManager

	notificationMgr NotificationManager
}
