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
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/http/api"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/http/middleware"
)

// Router represents http api router.
type Router struct {
	engine *gin.Engine
	deps   *depspkg.API

	loginAPI           *api.LoginAPI
	bootAPI            *api.BootAPI
	cmpAPI             *api.ComponentAPI
	integrationAPI     *api.IntegerationAPI
	orgAPI             *api.OrgAPI
	teamAPI            *api.TeamAPI
	userAPI            *api.UserAPI
	tagAPI             *api.TagAPI
	datasourceAPI      *api.DatasourceAPI
	datasourceQueryAPI *api.DatasourceQueryAPI

	dashboardAPI *api.DashboardAPI
	chartAPI     *api.ChartAPI

	notificationAPI *api.NotificationAPI
	alertRuleAPI    *api.AlertRuleAPI
}

// NewRouter creates a Router instance.
func NewRouter(engine *gin.Engine, deps *depspkg.API) *Router {
	return &Router{
		engine:             engine,
		deps:               deps,
		loginAPI:           api.NewLoginAPI(deps),
		bootAPI:            api.NewBootAPI(deps),
		cmpAPI:             api.NewComponentAPI(deps),
		integrationAPI:     api.NewIntegrationAPI(deps),
		orgAPI:             api.NewOrgAPI(deps),
		userAPI:            api.NewUserAPI(deps),
		teamAPI:            api.NewTeamAPI(deps),
		tagAPI:             api.NewTagAPI(deps),
		datasourceAPI:      api.NewDatasourceAPI(deps),
		datasourceQueryAPI: api.NewDatasourceQueryAPI(deps),

		dashboardAPI: api.NewDashboardAPI(deps),
		chartAPI:     api.NewChartAPI(deps),

		notificationAPI: api.NewNotificationAPI(deps),
		alertRuleAPI:    api.NewAlertRuleAPI(deps),
	}
}

// RegisterRouters registers all http api routers.
func (r *Router) RegisterRouters() {
	r.engine.Use(middleware.InitContext(r.deps))

	router := r.engine.Group(constant.APIV1)

	router.POST("/login", r.loginAPI.Login)
	router.GET("/logout", middleware.Authenticate(r.deps), r.loginAPI.Logout)
	router.GET("/boot", middleware.Authenticate(r.deps), r.bootAPI.Boot)

	router.GET("/integrations", middleware.Authenticate(r.deps), r.integrationAPI.GetIntegrations)
	router.GET("/tags", middleware.Authenticate(r.deps), r.tagAPI.FindTags)

	router.GET("/components",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.cmpAPI.LoadComponentTree)...)
	router.POST("/components",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.cmpAPI.CreateComponent)...)
	router.PUT("/components",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.cmpAPI.UpdateComponent)...)
	router.PUT("/components/sort",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.cmpAPI.SortComponents)...)
	router.DELETE("/components/:uid",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.cmpAPI.DeleteComponentByUID)...)
	router.PUT("/orgs/:uid/components",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Write, r.cmpAPI.SaveOrgComponents)...)
	router.GET("/orgs/:uid/components",
		middleware.Authorize(r.deps, accesscontrol.LinAccessResource, accesscontrol.Read, r.cmpAPI.GetOrgComponents)...)

	router.GET("/orgs/components",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.cmpAPI.GetComponentTreeByCurrentOrg)...)
	router.PUT("/orgs/components/roles",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.cmpAPI.UpdateRolesOfOrgComponent)...)

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

	router.PUT("/data/query",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceQueryAPI.DataQuery)...)
	router.PUT("/metadata/query",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.datasourceQueryAPI.MetadataQuery)...)

	// alerting
	r.registerAlertingRouters(router)
}

func (r *Router) registerAlertingRouters(router *gin.RouterGroup) {
	router.GET("/alerting/notification/settings",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Read, r.notificationAPI.GetAllNotificationSettings)...)
	router.PUT("/alerting/notification/settings",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.notificationAPI.SaveNotificationSetting)...)
	router.DELETE("/alerting/notification/settings/:type",
		middleware.Authorize(r.deps, accesscontrol.AdminAccessResource, accesscontrol.Write, r.notificationAPI.DeleteNotificationSettting)...)

	router.POST("/alerting/notifications",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationAPI.CreateNotification)...)
	router.PUT("/alerting/notifications",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationAPI.UpdateNotification)...)
	router.GET("/alerting/notifications",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationAPI.SearchNotifications)...)
	router.GET("/alerting/notifications/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.notificationAPI.GetNotificationByUID)...)

	router.PUT("/alerting/notifications/test",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.notificationAPI.Test)...)

	router.POST("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.CreateAlertRule)...)
	router.PUT("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.alertRuleAPI.UpdateAlertRule)...)
	router.DELETE("/alerting/rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.EditorAccessResource, accesscontrol.Write, r.chartAPI.DeleteChartByUID)...)
	router.GET("/alerting/rules/:uid",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleAPI.GetAlertRuleByUID)...)
	router.GET("/alerting/rules",
		middleware.Authorize(r.deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, r.alertRuleAPI.SearchAlertRules)...)
}
