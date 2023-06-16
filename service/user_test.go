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
	"gorm.io/gorm"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

func TestUserService_GetPreference(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	t.Run("get user's preference successfully", func(t *testing.T) {
		mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(10)).Return(nil)
		pref, err := srv.GetPreference(ctx)
		assert.NotNil(t, pref)
		assert.NoError(t, err)
	})
}

func TestUserService_SavePreference(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user's preference failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(10)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update user's preference failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(10)).Return(nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "user_id=?", int64(10)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update user's preference successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(10)).Return(nil)
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "user_id=?", int64(10)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "create user's preference successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(10)).Return(gorm.ErrRecordNotFound)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SavePreference(ctx, &model.Preference{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_ChangePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(10)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "old password invalid",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(10)).Return(nil)
			},
			wantErr: true,
		},
		{
			name: "change password successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(10)).DoAndReturn(func(m any, _ string, _ int64) error {
					user := m.(*model.User)
					user.Password = util.EncodePassword("12345", user.Salt)
					user.ID = 10
					return nil
				})
				mockDB.EXPECT().UpdateSingle(gomock.Any(), gomock.Any(), gomock.Any(), "id=?", int64(10)).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.ChangePassword(ctx, &model.ChangeUserPassword{
				OldPassword: "12345",
				NewPassword: "123456",
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_ResetPassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "reset password successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), gomock.Any(), gomock.Any(), "uid=?", "1234").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.ResetPassword(ctx, &model.ResetUserPassword{
				UserUID:  "1234",
				Password: "123456",
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserService_GetUserByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)

	srv := NewUserService(mockDB, nil)
	t.Run("get user successfully", func(t *testing.T) {
		mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
		user, err := srv.GetUserByUID(ctx, "1234")
		assert.NotNil(t, user)
		assert.NoError(t, err)
	})
}

func TestUserService_SearchUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "count failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "count 0",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), nil)
			},
			wantErr: false,
		},
		{
			name: "find failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "find successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(10), nil)
				mockDB.EXPECT().FindForPaging(gomock.Any(), 20, 10, "id desc", "name like ? or user_name like ? or email like ?",
					gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, _, err := srv.SearchUser(ctx, &model.SearchUserRequest{
				Query: "name",
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

func TestUserService_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "create user failure",
			prepare: func() {
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create user successfully",
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
			_, err := srv.CreateUser(ctx, &model.CreateUserRequest{})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserService_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update user successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "uid=?", "1234").Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateUser(ctx, &model.User{UID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserService_GetSignedUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1234)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "no org, get preference failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1234)).
					DoAndReturn(func(user *model.User, _ string, _ int64) error {
						user.ID = 1234
						return nil
					})
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(1234)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "has org, get user org fail",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1234)).
					DoAndReturn(func(user *model.User, _ string, _ int64) error {
						user.ID = 1234
						user.OrgID = 1
						return nil
					})
				mockDB.EXPECT().Get(gomock.Any(), "org_id=? and user_id=?", int64(1), int64(1234)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "has org, get org fail",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1234)).
					DoAndReturn(func(user *model.User, _ string, _ int64) error {
						user.ID = 1234
						user.OrgID = 1
						return nil
					})
				mockDB.EXPECT().Get(gomock.Any(), "org_id=? and user_id=?", int64(1), int64(1234)).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "has org, get preference ok",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1234)).
					DoAndReturn(func(user *model.User, _ string, _ int64) error {
						user.ID = 1234
						user.OrgID = 1
						return nil
					})
				mockDB.EXPECT().Get(gomock.Any(), "org_id=? and user_id=?", int64(1), int64(1234)).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "id=?", int64(1)).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "user_id=?", int64(1234)).Return(gorm.ErrRecordNotFound)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetSignedUser(ctx, 1234)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserService_DisableUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	mockDB.EXPECT().UpdateSingle(gomock.Any(), "is_disabled", true, "uid=?", "1234").Return(nil)
	err := srv.DisableUser(ctx, "1234")
	assert.NoError(t, err)
}

func TestUserService_EnableUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)
	mockDB.EXPECT().UpdateSingle(gomock.Any(), "is_disabled", false, "uid=?", "1234").Return(nil)
	err := srv.EnableUser(ctx, "1234")
	assert.NoError(t, err)
}

func TestUserSerivce_AddOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	orgSrv := NewMockOrgService(ctrl)
	srv := NewUserService(mockDB, orgSrv)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create user's org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create user's org successfully, but update user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "id=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create user's org successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Update(gomock.Any(), "id=?", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "create user's org successfully, user selected org",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").
					DoAndReturn(func(user *model.User, _, _ string) error {
						user.OrgID = 1
						return nil
					})
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.AddOrg(ctx, &model.UserOrgInfo{UserUID: "1234", OrgUID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_RemoveOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	orgSrv := NewMockOrgService(ctrl)
	srv := NewUserService(mockDB, orgSrv)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "delete user's org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "delete user's org successfully, but find other user's org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "delete user's org successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().Find(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(orgs *[]model.OrgUser, _ string, _ int64) error {
						*orgs = append(*orgs, model.OrgUser{})
						return nil
					})
				mockDB.EXPECT().UpdateSingle(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.RemoveOrg(ctx, &model.UserOrgInfo{UserUID: "1234", OrgUID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_UpdateOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	orgSrv := NewMockOrgService(ctrl)
	srv := NewUserService(mockDB, orgSrv)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update user's org successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "1234").Return(&model.Org{}, nil)
				mockDB.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateOrg(ctx, &model.UserOrgInfo{UserUID: "1234", OrgUID: "1234"})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_GetOrgListByUserUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get user failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org list failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org list successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetOrgListByUserUID(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestUserSerivce_SwitchOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewUserService(mockDB, nil)

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "check user org failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "user org not exist",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "switch org successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "1234").Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "org_id", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SwitchOrg(ctx, "1234")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
