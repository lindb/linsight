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
	// GetTeamByUID returns team by tean uid.
	GetTeamByUID(ctx context.Context, uid string) (*model.Team, error)

	// GetTeamMembers returns member list for team.
	GetTeamMembers(
		ctx context.Context,
		teamUID string,
		req *model.SearchTeamMemberRequest,
	) (rs []model.TeamMemberInfo, total int64, err error)
	// AddTeamMembers adds new team members.
	AddTeamMembers(ctx context.Context, teamUID string, members *model.AddTeamMember) error
	// UpdateTeamMember updates team member.
	UpdateTeamMember(ctx context.Context, teamUID string, member *model.UpdateTeamMember) error
	// RemoveTeamMember removes members from team.
	RemoveTeamMember(ctx context.Context, teamUID string, members *model.RemoveTeamMember) error
}

// teamService implements TeamService interface.
type teamService struct {
	db      dbpkg.DB
	userSrv UserService
}

// NewTeamService creates a TeamService instance.
func NewTeamService(db dbpkg.DB, userSrv UserService) TeamService {
	return &teamService{
		db:      db,
		userSrv: userSrv,
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
	count, err := srv.db.Count(&model.Team{}, where, params...)
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

// GetTeamByUID returns team by tean uid.
func (srv *teamService) GetTeamByUID(ctx context.Context, uid string) (*model.Team, error) {
	return srv.getTeamByUID(ctx, uid)
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

// AddTeamMembers adds new team members.
func (srv *teamService) AddTeamMembers(ctx context.Context, teamUID string, members *model.AddTeamMember) error {
	team, err := srv.getTeamByUID(ctx, teamUID)
	if err != nil {
		return err
	}
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		permission := members.Permission
		for _, userUID := range members.UserUIDs {
			// TODO: batch?
			user, err := srv.userSrv.GetUserByUID(ctx, userUID)
			if err != nil {
				return err
			}
			// check member if exist
			exist, err := tx.Exist(&model.TeamMember{},
				"org_id=? and team_id=? and user_id=?", team.OrgID, team.ID, user.ID)
			if err != nil {
				return err
			}
			if exist {
				// ignore if member exist
				continue
			}
			if err := tx.Create(&model.TeamMember{
				OrgID:      team.OrgID,
				TeamID:     team.ID,
				UserID:     user.ID,
				Permission: permission,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetTeamMembers returns member list for team.
func (srv *teamService) GetTeamMembers(
	ctx context.Context,
	teamUID string,
	req *model.SearchTeamMemberRequest,
) (rs []model.TeamMemberInfo, total int64, err error) {
	team, err := srv.getTeamByUID(ctx, teamUID)
	if err != nil {
		return nil, 0, err
	}
	conditions := []string{"org_id=?", "team_id=?"}
	params := []any{team.OrgID, team.ID}
	if req.User != "" {
		conditions = append(conditions, "user_id in (select id from users where (name like ? or user_name like ? or email like ?))")
		params = append(params, req.User+"%", req.User+"%", req.User+"%")
	}
	if len(req.Permissions) > 0 {
		conditions = append(conditions, "permission in ?")
		params = append(params, req.Permissions)
	}
	where := strings.Join(conditions, " and ")
	count, err := srv.db.Count(&model.TeamMember{}, where, params...)
	if err != nil {
		return nil, 0, err
	}
	if count == 0 {
		return nil, 0, nil
	}

	// reset params
	params = []any{team.OrgID, team.ID}
	sql := `
	select 
		u.uid as user_uid,u.name as name,u.user_name as user_name,tm.permission as permission  
	from users u,team_members tm 
		where u.id=tm.user_id and tm.org_id=? and tm.team_id=?`

	if req.User != "" {
		sql += " and (u.name like ? or u.user_name like ? or u.email like ?)"
		params = append(params, req.User+"%", req.User+"%", req.User+"%")
	}
	if len(req.Permissions) > 0 {
		sql += " and tm.permission in ?"
		params = append(params, req.Permissions)
	}
	offset := 0
	limit := 20
	if req.Offset > 0 {
		offset = req.Offset
	}
	if req.Limit > 0 {
		limit = req.Limit
	}
	sql += " order by tm.id desc limit ? offset ?"
	params = append(params, limit, offset)
	if err := srv.db.ExecRaw(&rs, sql, params...); err != nil {
		return nil, 0, err
	}
	return rs, count, nil
}

// RemoveTeamMember removes members from team.
func (srv *teamService) RemoveTeamMember(ctx context.Context, teamUID string, members *model.RemoveTeamMember) error {
	team, err := srv.getTeamByUID(ctx, teamUID)
	if err != nil {
		return err
	}
	return srv.db.Transaction(func(tx dbpkg.DB) error {
		for _, userUID := range members.UserUIDs {
			// TODO: batch?
			user, err := srv.userSrv.GetUserByUID(ctx, userUID)
			if err != nil {
				return err
			}
			if err := tx.Delete(&model.TeamMember{},
				"org_id=? and team_id=? and user_id=?", team.OrgID, team.ID, user.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateTeamMember updates team member.
func (srv *teamService) UpdateTeamMember(ctx context.Context, teamUID string, member *model.UpdateTeamMember) error {
	team, err := srv.getTeamByUID(ctx, teamUID)
	if err != nil {
		return err
	}
	user, err := srv.userSrv.GetUserByUID(ctx, member.UserUID)
	if err != nil {
		return err
	}
	return srv.db.UpdateSingle(&model.TeamMember{},
		"permission", member.Permission,
		"org_id=? and team_id=? and user_id=?", team.OrgID, team.ID, user.ID)
}
