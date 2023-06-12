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

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// TeamAPI represents team related api handlers.
type TeamAPI struct {
	deps *depspkg.API
}

// NewTeamAPI creates a TeamAPI instance.
func NewTeamAPI(deps *depspkg.API) *TeamAPI {
	return &TeamAPI{
		deps: deps,
	}
}

// CreateTeam creates a new team.
func (api *TeamAPI) CreateTeam(c *gin.Context) {
	var team model.Team
	if err := c.ShouldBind(&team); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	uid, err := api.deps.TeamSrv.CreateTeam(ctx, &team)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateTeam updates a team.
func (api *TeamAPI) UpdateTeam(c *gin.Context) {
	var team model.Team
	if err := c.ShouldBind(&team); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	err := api.deps.TeamSrv.UpdateTeam(ctx, &team)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Team updated")
}

// SearchTeams searches teams by given params.
func (api *TeamAPI) SearchTeams(c *gin.Context) {
	req := &model.SearchTeamRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	teams, total, err := api.deps.TeamSrv.SearchTeams(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total": total,
		"teams": teams,
	})
}

// DeleteTeamByUID deletes team by given uid.
func (api *TeamAPI) DeleteTeamByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.TeamSrv.DeleteTeamByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Team deleted")
}

// GetTeamByUID returns team by given uid.
func (api *TeamAPI) GetTeamByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	team, err := api.deps.TeamSrv.GetTeamByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, team)
}
