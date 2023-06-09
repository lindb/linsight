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
	"fmt"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestTeamService_SearchTeams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewTeamService(mockDB, nil)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "count failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "count 0",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			wantErr: false,
		},
		{
			name: "find failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "org_id=? and name like ?",
					gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.SearchTeams(ctx, &model.SearchTeamRequest{
				Name: "name",
				PagingParam: model.PagingParam{
					Limit:  10,
					Offset: 20,
				},
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_CreateTeam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewTeamService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create team failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create team successfully",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.CreateTeam(ctx, &model.Team{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_UpdateTeam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewTeamService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update team successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateTeam(ctx, &model.Team{UID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_DeleteTeamByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()

	srv := NewTeamService(mockDB, nil)
	t.Run("delete successfully", func(t *testing.T) {
		mockDB.EXPECT().Delete(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
		err := srv.DeleteTeamByUID(ctx, "1234")
		assert.NoError(t, err)
	})
}

func TestTeamService_GetTeamByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	srv := NewTeamService(mockDB, nil)
	t.Run("get team successfully", func(t *testing.T) {
		mockDB.EXPECT().Get(gomock.Any(), "uid=? and org_id=?", "1234", int64(12)).Return(nil)
		team, err := srv.GetTeamByUID(ctx, "1234")
		assert.NotNil(t, team)
		assert.NoError(t, err)
	})
}

func TestTeamService_AddTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()

	userSrv := NewMockUserService(ctrl)
	srv := NewTeamService(mockDB, userSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "chece exist failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "member exist",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(true, nil)
			},
			wantErr: false,
		},
		{
			name: "create failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create ok",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.AddTeamMembers(ctx, "team", &model.AddTeamMember{UserUIDs: []string{"1234"}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_RemoveTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()

	userSrv := NewMockUserService(ctrl)
	srv := NewTeamService(mockDB, userSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "remove failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "remove ok",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.RemoveTeamMember(ctx, "team", &model.RemoveTeamMember{UserUIDs: []string{"1234"}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_UpdateTeamMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	userSrv := NewMockUserService(ctrl)
	srv := NewTeamService(mockDB, userSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update member",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				userSrv.EXPECT().GetUserByUID(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateTeamMember(ctx, "team", &model.UpdateTeamMember{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestTeamService_GetTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	userSrv := NewMockUserService(ctrl)
	srv := NewTeamService(mockDB, userSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get team failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "count failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "no data",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			wantErr: false,
		},
		{
			name: "find data failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find data failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.GetTeamMembers(ctx, "team", &model.SearchTeamMemberRequest{
				User:        "user",
				Permissions: []model.PermissionType{model.PermissionAdmin},
				PagingParam: model.PagingParam{
					Limit:  10,
					Offset: 10,
				},
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
