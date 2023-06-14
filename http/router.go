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

type Router struct {
	engine *gin.Engine
	deps   *depspkg.API

	loginAPI           *api.LoginAPI
	bootAPI            *api.BootAPI
	orgAPI             *api.OrgAPI
	teamAPI            *api.TeamAPI
	navAPI             *api.NavAPI
	userAPI            *api.UserAPI
	datasourceAPI      *api.DatasourceAPI
	datasourceQueryAPI *api.DatasourceQueryAPI

	dashboardAPI *api.DashboardAPI
	chartAPI     *api.ChartAPI
}

func NewRouter(engine *gin.Engine, deps *depspkg.API) *Router {
	return &Router{
		engine:             engine,
		deps:               deps,
		loginAPI:           api.NewLoginAPI(deps),
		bootAPI:            api.NewBootAPI(deps),
		orgAPI:             api.NewOrgAPI(deps),
		userAPI:            api.NewUserAPI(deps),
		teamAPI:            api.NewTeamAPI(deps),
		navAPI:             api.NewNavAPI(deps),
		datasourceAPI:      api.NewDatasourceAPI(deps),
		datasourceQueryAPI: api.NewDatasourceQueryAPI(deps),

		dashboardAPI: api.NewDashboardAPI(deps),
		chartAPI:     api.NewChartAPI(deps),
	}
}

func (r *Router) RegisterRouters() {
	r.engine.Use(middleware.InitContext(r.deps))

	router := r.engine.Group(constant.APIV1)

	router.POST("/login", r.loginAPI.Login)
	router.GET("/logout", middleware.Authenticate(r.deps), r.loginAPI.Logout)
	router.GET("/boot", middleware.Authenticate(r.deps), r.bootAPI.Boot)

	router.GET("/orgs",
		middleware.Authorize(r.deps, accesscontrol.Org, accesscontrol.Read, r.orgAPI.SearchOrg)...)
	router.POST("/orgs",
		middleware.Authorize(r.deps, accesscontrol.Org, accesscontrol.Write, r.orgAPI.CreateOrg)...)
	router.PUT("/orgs",
		middleware.Authorize(r.deps, accesscontrol.Org, accesscontrol.Write, r.orgAPI.UpdateOrg)...)
	router.DELETE("/orgs/:uid",
		middleware.Authorize(r.deps, accesscontrol.Org, accesscontrol.Write, r.orgAPI.DeleteOrgByUID)...)
	router.GET("/orgs/:uid",
		middleware.Authorize(r.deps, accesscontrol.Org, accesscontrol.Read, r.orgAPI.GetOrgByUID)...)

	router.GET("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.Team, accesscontrol.Read, r.teamAPI.SearchTeams)...)
	router.GET("/org/teams/:uid",
		middleware.Authorize(r.deps, accesscontrol.Team, accesscontrol.Read, r.teamAPI.GetTeamByUID)...)
	router.POST("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.Team, accesscontrol.Write, r.teamAPI.CreateTeam)...)
	router.PUT("/org/teams",
		middleware.Authorize(r.deps, accesscontrol.Team, accesscontrol.Write, r.teamAPI.UpdateTeam)...)
	router.DELETE("/org/teams/:uid",
		middleware.Authorize(r.deps, accesscontrol.Team, accesscontrol.Write, r.teamAPI.DeleteTeamByUID)...)

	router.GET("/org/nav",
		middleware.Authorize(r.deps, accesscontrol.Nav, accesscontrol.Read, r.navAPI.GetNav)...)
	router.PUT("/org/nav",
		middleware.Authorize(r.deps, accesscontrol.Nav, accesscontrol.Write, r.navAPI.UpdateNav)...)

	router.POST("/user",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Write, r.userAPI.CreateUser)...)
	router.PUT("/user",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Write, r.userAPI.UpdateUser)...)
	router.GET("/users/:uid",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Read, r.userAPI.GetUser)...)
	router.GET("/user/preference",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Write, r.userAPI.GetPreference)...)
	router.PUT("/user/preference",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Write, r.userAPI.SavePreference)...)
	router.PUT("/user/password/change",
		middleware.Authorize(r.deps, accesscontrol.User, accesscontrol.Write, r.userAPI.ChangePassword)...)

	router.POST("/datasource",
		middleware.Authorize(r.deps, accesscontrol.Datasource, accesscontrol.Write, r.datasourceAPI.CreateDatasource)...)
	router.PUT("/datasource",
		middleware.Authorize(r.deps, accesscontrol.Datasource, accesscontrol.Write, r.datasourceAPI.UpdateDatasource)...)
	router.DELETE("/datasources/:uid",
		middleware.Authorize(r.deps, accesscontrol.Datasource, accesscontrol.Write, r.datasourceAPI.DeleteDatasourceByUID)...)
	router.GET("/datasources",
		middleware.Authorize(r.deps, accesscontrol.Datasource, accesscontrol.Read, r.datasourceAPI.GetDatasources)...)
	router.GET("/datasources/:uid",
		middleware.Authorize(r.deps, accesscontrol.Datasource, accesscontrol.Read, r.datasourceAPI.GetDatasourceByUID)...)

	// dashboard api
	router.POST("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Write, r.dashboardAPI.CreateDashboard)...)
	router.PUT("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Write, r.dashboardAPI.UpdateDashboard)...)
	router.DELETE("/dashboards/:uid",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Write, r.dashboardAPI.DeleteDashboardByUID)...)
	router.GET("/dashboards",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Read, r.dashboardAPI.SearchDashboards)...)
	router.GET("/dashboards/:uid",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Read, r.dashboardAPI.GetDashboardByUID)...)
	router.PUT("/dashboards/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Write, r.dashboardAPI.StarDashboard)...)
	router.DELETE("/dashboards/:uid/star",
		middleware.Authorize(r.deps, accesscontrol.Dashboard, accesscontrol.Write, r.dashboardAPI.UnstarDashboard)...)

	// chart repo api
	router.POST("/charts",
		middleware.Authorize(r.deps, accesscontrol.Chart, accesscontrol.Write, r.chartAPI.CreateChart)...)
	router.PUT("/charts",
		middleware.Authorize(r.deps, accesscontrol.Chart, accesscontrol.Write, r.chartAPI.UpdateChart)...)
	router.DELETE("/charts/:uid",
		middleware.Authorize(r.deps, accesscontrol.Chart, accesscontrol.Write, r.chartAPI.DeleteChartByUID)...)
	router.GET("/charts",
		middleware.Authorize(r.deps, accesscontrol.Chart, accesscontrol.Read, r.chartAPI.SearchCharts)...)

	router.PUT("/data/query",
		middleware.Authorize(r.deps, accesscontrol.DatasourceDataQuery, accesscontrol.Read, r.datasourceQueryAPI.DataQuery)...)
	router.PUT("/metadata/query",
		middleware.Authorize(r.deps, accesscontrol.DatasourceDataQuery, accesscontrol.Read, r.datasourceQueryAPI.MetadataQuery)...)
}
