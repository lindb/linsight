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

package service

import (
	"context"
	"strings"

	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

//go:generate mockgen -source=./team.go -destination=./team_mock.go -package=service

// TeamService represents team manager interface.
type TeamService interface {
	// SearchTeams searches the team by given params.
	SearchTeams(ctx context.Context, req *model.SearchTeamRequest) (rs []model.Team, total int64, err error)
	// CreateTeam creates a team, then returns tream uid, if fail returns error.
	CreateTeam(ctx context.Context, team *model.Team) (string, error)
	// UpdateTeam updates a team, if fail returns error.
	UpdateTeam(ctx context.Context, team *model.Team) error
	// DeleteTeamByUID deletes a team by team uid.
	DeleteTeamByUID(ctx context.Context, teamUID string) error
}

// teamService implements TeamService interface.
type teamService struct {
	db dbpkg.DB
}

// NewTeamService creates a TeamService instance.
func NewTeamService(db dbpkg.DB) TeamService {
	return &teamService{
		db: db,
	}
}

// SearchTeams searches the team by given params.
func (srv *teamService) SearchTeams(ctx context.Context,
	req *model.SearchTeamRequest,
) (rs []model.Team, total int64, err error) {
	conditions := []string{"org_id=?"}
	signedUser := util.GetUser(ctx)
	params := []any{signedUser.Org.ID}
	if req.Name != "" {
		conditions = append(conditions, "name like ?")
		params = append(params, req.Name+"%")
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.Chart{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}
	if err := srv.db.FindForPaging(&rs, offset, limit, "id desc", where, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// CreateTeam creates a team, then returns tream uid, if fail returns error.
func (srv *teamService) CreateTeam(ctx context.Context, team *model.Team) (string, error) {
	team.UID = uuid.GenerateShortUUID()
	// set team org/user info
	user := util.GetUser(ctx)
	team.OrgID = user.Org.ID
	userID := user.User.ID
	team.CreatedBy = userID
	team.UpdatedBy = userID
	if err := srv.db.Create(team); err != nil {
		return "", err
	}
	return team.UID, nil
}

// UpdateTeam updates a team, if fail returns error.
func (srv *teamService) UpdateTeam(ctx context.Context, team *model.Team) error {
	teamFromDB, err := srv.getTeamByUID(ctx, team.UID)
	if err != nil {
		return err
	}
	// TODO: check name if exist?
	user := util.GetUser(ctx)
	teamFromDB.Name = team.Name
	teamFromDB.UpdatedBy = user.User.ID
	return srv.db.Update(teamFromDB, "uid=? and org_id=?", team.UID, user.Org.ID)
}

// DeleteTeamByUUID deletes a team by team uid.
func (srv *teamService) DeleteTeamByUID(ctx context.Context, teamUID string) error {
	signedUser := util.GetUser(ctx)
	orgID := signedUser.Org.ID
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		// delete team
		// TODO: delete other?
		return tx.Delete(&model.Team{}, "uid=? and org_id=?", teamUID, orgID)
	})
}

// getTeamByUID returns the team by uid.
func (srv *teamService) getTeamByUID(ctx context.Context, uid string) (*model.Team, error) {
	rs := &model.Team{}
	signedUser := util.GetUser(ctx)
	if err := srv.db.Get(rs, "uid=? and org_id=?", uid, signedUser.Org.ID); err != nil {
		return nil, err
	}
	return rs, nil
}
