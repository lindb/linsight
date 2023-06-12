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

package api

import (
	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
)

// BootAPI represents boot information related api handlers.
type BootAPI struct {
	deps *depspkg.API
}

// NewBootAPI creates a BootAPI instance.
func NewBootAPI(deps *depspkg.API) *BootAPI {
	return &BootAPI{
		deps: deps,
	}
}

// Boot gets boot information after signed in.
func (api *BootAPI) Boot(c *gin.Context) {
	datasources, err := api.deps.DatasourceSrv.GetDatasources(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	signedUser := util.GetUser(c.Request.Context())
	// FIXME: need modify nav item data based on use setting
	httppkg.OK(c, &model.BootData{
		Home:        "/dashboards",
		User:        *signedUser,
		Datasources: datasources,
		NavTree: []model.NavItem{
			{
				Text: "Explore",
				Icon: "explore",
				Path: "/explore",
			},
			{
				Text: "Dashboard",
				Icon: "dashboard",
				Children: []model.NavItem{
					{
						Text: "Dashboard List",
						Path: "/dashboards",
						Icon: "list",
					},
					{
						Text: "New Dashboard",
						Path: "/dashboard",
						Icon: "dashboard-add",
					},
					{
						Text: "Chart Repository",
						Path: "/charts",
						Icon: "repo",
					},
				},
			},
			{
				Text: "Integration",
				Icon: "integrations",
				Path: "/integrations",
			},
			{
				Text: "APM",
				Icon: "tracing",
				Children: []model.NavItem{
					{
						Text: "Service",
						Path: "/observe/apm/service",
						Icon: "repo",
						Props: map[string]any{
							"dashboard": "cgV623b4k",
						},
					},
					{
						Text: "Dashboard",
						Path: "/observe/apm/database",
						Icon: "repo",
						Props: map[string]any{
							"dashboard": "cgV623b4k4",
						},
					},
				},
			},
			{
				Text: "Setting",
				Icon: "setting",
				Path: "/setting",
				Children: []model.NavItem{
					{
						Text: "Datasource",
						Path: "/setting/datasources",
						Icon: "datasource",
					},
					{
						Text: "User",
						Path: "/setting/org/users",
						Icon: "user",
					},
					{
						Text: "Team",
						Path: "/setting/org/teams",
						Icon: "team",
					},
				},
			},
		},
	})
}
