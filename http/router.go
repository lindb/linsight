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

package http

import (
	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/accesscontrol"
	aimodule "github.com/lindb/linsight/ai"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/http/api"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/http/middleware"
	"github.com/lindb/linsight/integrations"
)

// Router represents http api router.
type Router struct {
	engine *gin.Engine
	deps   *depspkg.API

	loginAPI           *api.LoginAPI
	bootAPI            *api.BootAPI
	menuAPI            *api.MenuAPI
	orgAPI             *api.OrgAPI
	teamAPI            *api.TeamAPI
	userAPI            *api.UserAPI
	tagAPI             *api.TagAPI
	datasourceAPI      *api.DatasourceAPI
	datasourceQueryAPI *api.DatasourceQueryAPI

	dashboardAPI        *api.DashboardAPI
	dashboardVersionAPI *api.DashboardVersionAPI
	chartAPI            *api.ChartAPI

	notificationAPI         *api.NotificationAPI
	alertRuleGroupAPI       *api.AlertRuleGroupAPI
	alertRuleAPI            *api.AlertRuleAPI
	alertEventAPI           *api.AlertEventAPI
	alertRouteAPI           *api.AlertRouteAPI
	contactPointAPI         *api.ContactPointAPI
	alertSilenceAPI         *api.AlertSilenceAPI
	alertInhibitAPI         *api.AlertInhibitAPI
	notificationTemplateAPI *api.NotificationTemplateAPI
}

// NewRouter creates a Router instance.
func NewRouter(engine *gin.Engine, deps *depspkg.API) *Router {
	return &Router{
		engine:             engine,
		deps:               deps,
		loginAPI:           api.NewLoginAPI(deps),
		bootAPI:            api.NewBootAPI(deps),
		menuAPI:            api.NewMenuAPI(deps),
		orgAPI:             api.NewOrgAPI(deps),
		userAPI:            api.NewUserAPI(deps),
		teamAPI:            api.NewTeamAPI(deps),
		tagAPI:             api.NewTagAPI(deps),
		datasourceAPI:      api.NewDatasourceAPI(deps),
		datasourceQueryAPI: api.NewDatasourceQueryAPI(deps),

		dashboardAPI:        api.NewDashboardAPI(deps),
		dashboardVersionAPI: api.NewDashboardVersionAPI(deps),
		chartAPI:            api.NewChartAPI(deps),

		notificationAPI:         api.NewNotificationAPI(deps),
		alertRuleGroupAPI:       api.NewAlertRuleGroupAPI(deps),
		alertRuleAPI:            api.NewAlertRuleAPI(deps),
		alertEventAPI:           api.NewAlertEventAPI(deps),
		alertRouteAPI:           api.NewAlertRouteAPI(deps),
		contactPointAPI:         api.NewContactPointAPI(deps),
		alertSilenceAPI:         api.NewAlertSilenceAPI(deps),
		alertInhibitAPI:         api.NewAlertInhibitAPI(deps),
		notificationTemplateAPI: api.NewNotificationTemplateAPI(deps),
	}
}

// RegisterRouters registers all http api routers.
func (r *Router) RegisterRouters() {
	r.engine.Use(middleware.InitContext(r.deps))

	router := r.engine.Group(constant.APIV1)

	router.POST("/login", r.loginAPI.Login)
	router.GET("/logout", middleware.Authenticate(r.deps), r.loginAPI.Logout)
	router.GET("/boot", middleware.Authenticate(r.deps), r.bootAPI.Boot)

	router.GET("/tags", middleware.Authenticate(r.deps), r.tagAPI.FindTags)

	router.GET("/menus",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.menuAPI.LoadMenuTree)...)
	router.POST("/menus",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.menuAPI.CreateMenu)...)
	router.PUT("/menus",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.menuAPI.UpdateMenu)...)
	router.PUT("/menus/sort",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.menuAPI.SortMenus)...)
	router.DELETE("/menus/:uid",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.menuAPI.DeleteMenuByUID)...)
	router.PUT("/orgs/:uid/menus",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.menuAPI.SaveOrgMenus)...)
	router.GET("/orgs/:uid/menus",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.menuAPI.GetOrgMenus)...)

	router.GET("/orgs/menus",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.menuAPI.GetMenuTreeByCurrentOrg)...)
	router.PUT("/orgs/menus/roles",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.menuAPI.UpdateRolesOfOrgMenus)...)

	router.GET("/orgs",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.orgAPI.SearchOrg)...)
	router.POST("/orgs",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.orgAPI.CreateOrg)...)
	router.PUT("/orgs",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.orgAPI.UpdateOrg)...)
	router.DELETE("/orgs/:uid",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.orgAPI.DeleteOrgByUID)...)
	router.GET("/orgs/:uid",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.orgAPI.GetOrgByUID)...)

	router.GET("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.teamAPI.SearchTeams)...)
	router.GET("/org/teams/:uid",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.teamAPI.GetTeamByUID)...)
	router.POST("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.CreateTeam)...)
	router.PUT("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.UpdateTeam)...)
	router.DELETE("/org/teams/:uid",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.DeleteTeamByUID)...)
	router.POST("/org/teams/:uid/members",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.AddTeamMembers)...)
	router.PUT("/org/teams/:uid/members",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.UpdateTeamMember)...)
	router.GET("/org/teams/:uid/members",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.teamAPI.GetTeamMembers)...)
	router.PUT("/org/teams/:uid/members/remove",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.teamAPI.RemoveTeamMember)...)

	router.GET("/org/users",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.orgAPI.GetUserListForSignedOrg)...)

	router.GET("/user/orgs",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.orgAPI.GetOrgListForSignedUser)...)

	router.POST("/users",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.userAPI.CreateUser)...)
	router.PUT("/users",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.userAPI.UpdateUser)...)
	router.GET("/users",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.userAPI.SearchUser)...)
	router.GET("/users/:uid",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.userAPI.GetUserByUID)...)
	router.GET("/users/:uid/orgs", middleware.Authenticate(r.deps), r.userAPI.GetOrgListByUserUID)
	router.POST("/users/:uid/orgs",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.userAPI.AddOrg)...)
	router.PUT("/users/:uid/orgs",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.userAPI.UpdateOrg)...)
	router.PUT("/user/orgs/switch/:orgUid", middleware.Authenticate(r.deps), r.userAPI.SwitchOrg)
	router.DELETE("/users/:uid/orgs/:orgUid",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.userAPI.RemoveOrg)...)
	router.PUT("/users/:uid/disable",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.userAPI.DisableUserByUID)...)
	router.PUT("/users/:uid/enable",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.userAPI.EnableUserByUID)...)
	// preference
	router.GET("/user/preference", middleware.Authenticate(r.deps), r.userAPI.GetPreference)
	router.PUT("/user/preference", middleware.Authenticate(r.deps), r.userAPI.SavePreference)
	// password
	router.PUT("/user/password/change", middleware.Authenticate(r.deps), r.userAPI.ChangePassword)
	router.PUT("/user/password/reset",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.userAPI.ResetPassword)...)

	router.POST("/datasource",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.datasourceAPI.CreateDatasource)...)
	router.PUT("/datasource",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.datasourceAPI.UpdateDatasource)...)
	router.DELETE("/datasources/:uid",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.datasourceAPI.DeleteDatasourceByUID)...)
	router.GET("/datasources",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceAPI.GetDatasources)...)
	router.GET("/datasources/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceAPI.GetDatasourceByUID)...)

	// dashboard api
	router.POST("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.dashboardAPI.CreateDashboard)...)
	router.PUT("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.dashboardAPI.UpdateDashboard)...)
	router.DELETE("/dashboards/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.dashboardAPI.DeleteDashboardByUID)...)
	router.GET("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.dashboardAPI.SearchDashboards)...)
	router.GET("/dashboards/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.dashboardAPI.GetDashboardByUID)...)
	router.PUT("/dashboards/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.dashboardAPI.StarDashboard)...)
	router.DELETE("/dashboards/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.dashboardAPI.UnstarDashboard)...)

	// dashboard version api
	router.GET("/dashboards/:uid/versions",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.dashboardVersionAPI.GetDashboardVersions)...)
	router.GET("/dashboards/:uid/versions/:version",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.dashboardVersionAPI.GetDashboardVersion)...)

	// chart repo api
	router.POST("/charts",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.chartAPI.CreateChart)...)
	router.PUT("/charts",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.chartAPI.UpdateChart)...)
	router.DELETE("/charts/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.chartAPI.DeleteChartByUID)...)
	router.GET("/charts/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.chartAPI.GetChartByUID)...)
	router.GET("/charts",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.chartAPI.SearchCharts)...)
	router.GET("/charts/:uid/dashboards",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.chartAPI.GetDashboardsByChartUID)...)
	router.DELETE("/charts/:uid/dashboards/:dashboardUID",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.chartAPI.UnlinkChartFromDashboard)...)
	router.PUT("/charts/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.chartAPI.StarChart)...)
	router.DELETE("/charts/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, r.chartAPI.UnstarChart)...)

	router.PUT("/data/query",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceQueryAPI.DataQuery)...)
	router.PUT("/metadata/query",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceQueryAPI.MetadataQuery)...)

	// alerting
	r.registerAlertingRouters(router)

	// ai agent (no-op when agent module is disabled)
	aimodule.RegisterRoutes(router, r.deps)

	// integrations knowledge base (no-op when integrations dir is not configured)
	integrations.RegisterRoutes(router, r.deps)
}

func (r *Router) registerAlertingRouters(router *gin.RouterGroup) {
	router.GET("/alerting/notification/settings",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.notificationAPI.GetAllNotificationSettings)...)
	router.PUT("/alerting/notification/settings",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.notificationAPI.SaveNotificationSetting)...)
	router.DELETE("/alerting/notification/settings/:type",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.notificationAPI.DeleteNotificationSettting)...)

	router.PUT("/alerting/notifications/test",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationAPI.Test)...)

	// notifier plugin definitions (replaces the removed boot-data Notifications field)
	router.GET("/alerting/notifiers",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.contactPointAPI.GetNotifierPlugins)...)

	// alert rules
	router.POST("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.CreateAlertRule)...)
	router.PUT("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.UpdateAlertRule)...)
	// BUG FIX: was mistakenly calling chartAPI.DeleteChartByUID
	router.DELETE("/alerting/rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.DeleteAlertRuleByUID)...)
	router.GET("/alerting/rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleAPI.GetAlertRuleByUID)...)
	router.GET("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleAPI.SearchAlertRules)...)
	router.POST("/alerting/rules/dry-run",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.DryRunAlertRule)...)

	// alert rule groups (created implicitly via rules; no POST endpoint)
	router.GET("/alerting/rule-groups",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleGroupAPI.SearchAlertRuleGroups)...)
	router.PUT("/alerting/rule-groups",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleGroupAPI.UpdateAlertRuleGroup)...)
	router.GET("/alerting/rule-groups/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleGroupAPI.GetAlertRuleGroupByUID)...)
	router.DELETE("/alerting/rule-groups/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleGroupAPI.DeleteAlertRuleGroupByUID)...)

	// alert events
	router.GET("/alerting/events",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertEventAPI.SearchAlertEvents)...)
	router.GET("/alerting/events/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertEventAPI.GetAlertEventByUID)...)

	// alert routes
	router.GET("/alerting/routes",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRouteAPI.GetAlertRouteTree)...)
	router.GET("/alerting/routes/search",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRouteAPI.SearchAlertRoutes)...)
	router.PUT("/alerting/routes/sort",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRouteAPI.SortAlertRoutes)...)
	router.GET("/alerting/routes/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRouteAPI.GetAlertRouteByUID)...)
	router.POST("/alerting/routes",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRouteAPI.CreateAlertRoute)...)
	router.PUT("/alerting/routes",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRouteAPI.UpdateAlertRoute)...)
	router.DELETE("/alerting/routes/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRouteAPI.DeleteAlertRouteByUID)...)

	// contact points
	router.GET("/alerting/contact-points",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.contactPointAPI.SearchContactPoints)...)
	router.GET("/alerting/contact-points/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.contactPointAPI.GetContactPointByUID)...)
	router.POST("/alerting/contact-points",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.contactPointAPI.CreateContactPoint)...)
	router.PUT("/alerting/contact-points",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.contactPointAPI.UpdateContactPoint)...)
	router.DELETE("/alerting/contact-points/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.contactPointAPI.DeleteContactPointByUID)...)
	router.PUT("/alerting/contact-points/test",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.contactPointAPI.TestContactPoint)...)
	router.POST("/alerting/contact-points/test/channel",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.contactPointAPI.TestNotificationChannel)...)

	// alert silences
	router.GET("/alerting/silences",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertSilenceAPI.SearchAlertSilences)...)
	router.GET("/alerting/silences/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertSilenceAPI.GetAlertSilenceByUID)...)
	router.POST("/alerting/silences",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertSilenceAPI.CreateAlertSilence)...)
	router.PUT("/alerting/silences",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertSilenceAPI.UpdateAlertSilence)...)
	router.DELETE("/alerting/silences/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertSilenceAPI.DeleteAlertSilenceByUID)...)

	// alert inhibit rules
	router.GET("/alerting/inhibit-rules",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertInhibitAPI.SearchAlertInhibitRules)...)
	router.GET("/alerting/inhibit-rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertInhibitAPI.GetAlertInhibitRuleByUID)...)
	router.POST("/alerting/inhibit-rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertInhibitAPI.CreateAlertInhibitRule)...)
	router.PUT("/alerting/inhibit-rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertInhibitAPI.UpdateAlertInhibitRule)...)
	router.DELETE("/alerting/inhibit-rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertInhibitAPI.DeleteAlertInhibitRuleByUID)...)

	// notification templates
	router.GET("/alerting/templates",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationTemplateAPI.SearchNotificationTemplates)...)
	router.GET("/alerting/templates/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationTemplateAPI.GetNotificationTemplateByUID)...)
	router.POST("/alerting/templates",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationTemplateAPI.CreateNotificationTemplate)...)
	router.PUT("/alerting/templates",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationTemplateAPI.UpdateNotificationTemplate)...)
	router.DELETE("/alerting/templates/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationTemplateAPI.DeleteNotificationTemplateByUID)...)
	router.POST("/alerting/templates/preview",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationTemplateAPI.PreviewNotificationTemplate)...)
	router.GET("/alerting/templates/definitions",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationTemplateAPI.GetTemplateDefinitions)...)

	// notification template examples (read-only; built-in static examples loaded from YAML)
	router.GET("/alerting/template-examples",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationTemplateAPI.SearchNotificationTemplateExamples)...)
}
