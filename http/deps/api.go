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

package deps

import (
	"github.com/lindb/linsight/alerting/manager"
	"github.com/lindb/linsight/alerting/schedule"
	"github.com/lindb/linsight/config"
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

// API holds all service dependencies injected into HTTP handlers.
type API struct {
	Config *config.Server

	OrgSrv         service.OrgService
	UserSrv        service.UserService
	UserCacheSrv   service.UserCacheService
	TeamSrv        service.TeamService
	MenuSrv        service.MenuService
	IntegrationSrv service.IntegrationService

	DatasourceSrv   service.DatasourceService
	AuthenticateSrv service.AuthenticateService
	AuthorizeSrv    service.AuthorizeService

	TagSrv              service.TagService
	DashboardSrv        service.DashboardService
	DashboardVersionSrv service.DashboardVersionService
	ChartSrv            service.ChartService

	NotificationSrv      service.NotificationService
	AlertRuleGroupSrv    service.AlertRuleGroupService
	AlertRuleSrv         service.AlertRuleService

	// new alerting services
	AlertEventSrv                  service.AlertEventService
	AlertRouteSrv                  service.AlertRouteService
	ContactPointSrv                service.ContactPointService
	ContactPointStatusSrv          service.ContactPointStatusService // delivery health per channel
	AlertSilenceSrv                service.AlertSilenceService
	AlertInhibitRuleSrv            service.AlertInhibitRuleService
	NotificationTemplateSrv        service.NotificationTemplateService
	NotificationTemplateExampleSrv service.NotificationTemplateExampleService

	AlertMgr manager.AlertManager

	DatasourceMgr datasource.Manager

	DryRunner schedule.DryRunner
}
