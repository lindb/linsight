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

func TestComponentService_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgSrv := NewMockOrgService(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	mockDB := db.NewMockDB(ctrl)

	srv := &componentService{
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
			name: "unmarshal cmp failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByName(gomock.Any(), constant.AdminOrgName).Return(nil, nil)
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
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

func TestComponentService_saveComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgSrv := NewMockOrgService(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	mockDB := db.NewMockDB(ctrl)

	srv := &componentService{
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
			name: "get cmp failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create cmp failure",
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
			name: "create org. cmp failure",
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
			name: "save child cmp failure",
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
			err := srv.saveComponent(&model.Component{
				Children: []*model.Component{{}},
			}, 123)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_LoadComponentTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewComponentService(mockDB, nil, nil)

	t.Run("load failure", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(fmt.Errorf("err"))
		tree, err := srv.LoadComponentTree(context.TODO())
		assert.Error(t, err)
		assert.Nil(t, tree)
	})

	t.Run("load successfully", func(t *testing.T) {
		mockDB.EXPECT().Find(gomock.Any()).Return(nil)
		tree, err := srv.LoadComponentTree(context.TODO())
		assert.NoError(t, err)
		assert.Nil(t, tree)
	})
}

func TestComponentService_UpdateComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewComponentService(mockDB, nil, nil)

	cmp := &model.Component{UID: "1234"}
	mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=?", "1234").Return(fmt.Errorf("err"))
	err := srv.UpdateComponent(ctx, cmp)
	assert.Error(t, err)
}

func TestComponentService_CreateComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewComponentService(mockDB, nil, nil)
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
			name: "create cmp failure",
			prepare: func() {
				mockDB.EXPECT().Exist(gomock.Any(), "uid=?", "123").Return(true, nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create cmp successfully",
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
			_, err := srv.CreateComponent(ctx, &model.Component{
				ParentUID: "123",
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_DeleteComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	srv := NewComponentService(mockDB, nil, nil)
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
			name: "delete cmp successfully",
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
			err := srv.DeleteComponentByUID(ctx, "abc")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_SortComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	srv := NewComponentService(mockDB, nil, nil)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "sort cmps failure",
			prepare: func() {
				mockDB.EXPECT().Updates(gomock.Any(), gomock.Any(), "uid=?", "abc").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "sort cmps successfully",
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
			err := srv.SortComponents(ctx, model.Components{{UID: "abc"}, {UID: "abc"}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_GetComponentTreeByCurrentOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewComponentService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "find cmps failure",
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
			name: "get org's cmps successfully",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), int64(12)).DoAndReturn(func(cmps *model.Components, _ string, _ int64) error {
					*cmps = append(*cmps, &model.Component{})
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
			_, err := srv.GetComponentTreeByCurrentOrg(ctx)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_GetOrgComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewComponentService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get org. cmps failure",
			prepare: func() {
				mockDB.EXPECT().ExecRaw(gomock.Any(), gomock.Any(), "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get org's cmps successfully",
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
			_, err := srv.GetOrgComponents(ctx, "123")
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_UpdateRolesOfOrgComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	authorizeSrv := NewMockAuthorizeService(ctrl)
	srv := NewComponentService(mockDB, nil, authorizeSrv)
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "get cmp failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update role for org's cmp failure",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "role",
					accesscontrol.RoleAdmin, gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "update role for org's cmp successfully",
			prepare: func() {
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().UpdateSingle(gomock.Any(), "role",
					accesscontrol.RoleAdmin, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.UpdateRolesOfOrgComponent(ctx,
				[]model.OrgComponentInfo{{ComponentUID: "123", Role: accesscontrol.RoleAdmin}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestComponentService_SaveOrgComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().Transaction(gomock.Any()).DoAndReturn(func(fn func(tx db.DB) error) error {
		return fn(mockDB)
	}).AnyTimes()
	authorizeSrv := NewMockAuthorizeService(ctrl)
	orgSrv := NewMockOrgService(ctrl)
	srv := NewComponentService(mockDB, orgSrv, authorizeSrv)
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
			name: "delete org cmps failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "get cmp failure",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "create org cmp failure",
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
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Component).Return(fmt.Errorf("err"))
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
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Component).Return(nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "save org's cmp successfully",
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				mockDB.EXPECT().Delete(gomock.Any(), "org_id=?", gomock.Any()).Return(nil)
				mockDB.EXPECT().Get(gomock.Any(), "uid=?", "123").Return(nil)
				mockDB.EXPECT().Create(gomock.Any()).Return(nil)
				authorizeSrv.EXPECT().RemoveResourcePoliciesByCategory(gomock.Any(), accesscontrol.Component).Return(nil)
				authorizeSrv.EXPECT().AddResourcePolicy(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.SaveOrgComponents(ctx, "123", []model.OrgComponentInfo{{ComponentUID: "123", Role: accesscontrol.RoleAdmin}})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
