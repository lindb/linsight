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
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lindb/common/pkg/encoding"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lindb/linsight/accesscontrol"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
)

func TestMenuService_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgSrv := NewMockOrgService(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	mockDB := db.NewMockDB(ctrl)

	srv := &menService{
		orgSrv:       orgSrv,
		authorizeSrv: authorizeSrv,
		db:           mockDB,
	}
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get main org. failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByName(gomock.Any(), constant.AdminOrgName).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "unmarshal menu failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByName(gomock.Any(), constant.AdminOrgName).Return(nil, nil)
				jsonUnmarshalFn = func(_ []byte, _ any) error {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "initialize failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByName(gomock.Any(), constant.AdminOrgName).Return(&model.Org{}, nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "initialize successfully",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByName(gomock.Any(), constant.AdminOrgName).Return(&model.Org{}, nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(nil).AnyTimes()
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
			}()
			tt.prepare()
			err := srv.Initialize()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_saveMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgSrv := NewMockOrgService(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	mockDB := db.NewMockDB(ctrl)

	srv := &menService{
		orgSrv:       orgSrv,
		authorizeSrv: authorizeSrv,
		db:           mockDB,
	}
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(gorm.ErrRecordNotFound)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "check exist failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(gorm.ErrRecordNotFound)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create org. menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "add acl policy failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "save child menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Exist(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.saveMenu(&model.Menu{
				Children: []*model.Menu{{}},
			}, 123)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_LoadMenuTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewMenuService(mockDB, nil, nil)

	t.Run("load failure", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(fmt.Errorf("err"))
		tree, err := srv.LoadMenuTree(context.TODO())
		assert.Error(t, err)
		assert.Nil(t, tree)
	})

	t.Run("load successfully", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(nil)
		tree, err := srv.LoadMenuTree(context.TODO())
		assert.NoError(t, err)
		assert.Nil(t, tree)
	})
}

func TestMenuService_UpdateMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewMenuService(mockDB, nil, nil)

	menu := &model.Menu{UID: "1234"}
	mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
	err := srv.UpdateMenu(ctx, menu)
	assert.Error(t, err)
}

func TestMenuService_CreateMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewMenuService(mockDB, nil, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "check parent failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "parent not exist",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", "123").Return(false, nil)
			},
			wantErr: true,
		},
		{
			name: "create menu failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", "123").Return(true, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create menu successfully",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", "123").Return(true, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.CreateMenu(ctx, &model.Menu{
				ParentUID: "123",
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_DeleteMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewMenuService(mockDB, nil, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "check check failure",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "parent_uid=?", "abc").Return(int64(0), fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "has children",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "parent_uid=?", "abc").Return(int64(10), nil)
			},
			wantErr: true,
		},
		{
			name: "delete menu successfully",
			prepare: func() {
				mockDB.EXPECT().Count(gomock.Any(), "parent_uid=?", "abc").Return(int64(0), nil)
				mockDB.EXPECT().Delete(gomock.Any(), "uid=?", "abc").Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.DeleteMenuByUID(ctx, "abc")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_SortMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewMenuService(mockDB, nil, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "sort menus failure",
			prepare: func() {
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=?", "abc").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "sort menus successfully",
			prepare: func() {
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=?", "abc").Return(nil).MaxTimes(2)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SortMenus(ctx, model.Menus{{UID: "abc"}, {UID: "abc"}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_GetMenuTreeByCurrentOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewMenuService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "find menus failure",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "check acl failure",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).Return(nil)
				authorizeSrv.EXPECT().CheckResourcesACL(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org's menus successfully",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).DoAndReturn(func(menus *model.Menus, _ string, _ int64) error {
					*menus = append(*menus, &model.Menu{})
					return nil
				})
				authorizeSrv.EXPECT().CheckResourcesACL(gomock.Any()).Return([]bool{true}, nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetMenuTreeByCurrentOrg(ctx)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_GetOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewMenuService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get org. menus failure",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org's menus successfully",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), "123").Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			_, err := srv.GetOrgMenus(ctx, "123")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_UpdateRolesOfOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewMenuService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update role for org's menu failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "role",
					accesscontrol.RoleAdmin, "org_id=? and menu_id=?", int64(12),
					gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update acl role failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "role",
					accesscontrol.RoleAdmin, "org_id=? and menu_id=?", int64(12),
					gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().UpdateResourceRole(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update role for org's menu successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "role",
					accesscontrol.RoleAdmin, "org_id=? and menu_id=?", int64(12),
					gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().UpdateResourceRole(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateRolesOfOrgMenus(ctx,
				[]model.OrgMenuInfo{{MenuUID: "123", Role: accesscontrol.RoleAdmin}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestMenuService_SaveOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	authorizeSrv := NewMockAuthorizeService(ctrl)
	orgSrv := NewMockOrgService(ctrl)
	srv := NewMenuService(mockDB, orgSrv, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get org failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "delete org menus failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get menu failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create org menu failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "remove acl failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Menu).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "add acl failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Menu).Return(nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "save org's menu successfully",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Menu).Return(nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SaveOrgMenus(ctx, "123", []model.OrgMenuInfo{{MenuUID: "123", Role: accesscontrol.RoleAdmin}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
