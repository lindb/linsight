package schedule

import (
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

type Deps struct {
	Opts            *Options
	DatasourceSrv   service.DatasourceService
	AlertRuleSrv    service.AlertRuleService
	NotificationSrv service.NotificationService
	DatasourceMgr   datasource.Manager

	notificationMgr NotificationManager
}
