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

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/accesscontrol"
	modelpkg "github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/db"
	casbinmock "github.com/lindb/linsight/service/casbin"
)

func TestAuthorizeService_New(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		newAdapterByDBWithCustomTableFn = gormadapter.NewAdapterByDBWithCustomTable
		newModelFromStringFn = model.NewModelFromString
		newEnforcerFn = casbin.NewEnforcer
		ctrl.Finish()
	}()
	mockDB := db.NewMockDB(ctrl)
	mockDB.EXPECT().RawDB().Return(nil).AnyTimes()

	assert.Panics(t, func() {
		newAdapterByDBWithCustomTableFn = func(_ *gorm.DB, _ interface{}, _ ...string) (*gormadapter.Adapter, error) {
			return nil, fmt.Errorf("err")
		}
		_ = NewAuthorizeService(mockDB)
		assert.True(t, false)
	})
	assert.Panics(t, func() {
		newAdapterByDBWithCustomTableFn = func(_ *gorm.DB, _ interface{}, _ ...string) (*gormadapter.Adapter, error) {
			return nil, nil
		}
		newModelFromStringFn = func(_ string) (model.Model, error) {
			return nil, fmt.Errorf("err")
		}
		NewAuthorizeService(mockDB)
		assert.True(t, false)
	})
	assert.Panics(t, func() {
		newAdapterByDBWithCustomTableFn = func(_ *gorm.DB, _ interface{}, _ ...string) (*gormadapter.Adapter, error) {
			return nil, nil
		}
		newModelFromStringFn = func(_ string) (model.Model, error) {
			return nil, nil
		}
		newEnforcerFn = func(_ ...interface{}) (*casbin.Enforcer, error) {
			return nil, fmt.Errorf("err")
		}
		NewAuthorizeService(mockDB)
		assert.True(t, false)
	})
	newAdapterByDBWithCustomTableFn = func(_ *gorm.DB, _ interface{}, _ ...string) (*gormadapter.Adapter, error) {
		return nil, nil
	}
	newModelFromStringFn = func(_ string) (model.Model, error) {
		return nil, nil
	}
	newEnforcerFn = func(_ ...interface{}) (*casbin.Enforcer, error) {
		return nil, nil
	}
	srv := NewAuthorizeService(mockDB)
	assert.NotNil(t, srv)
}

func TestAuthorizeService_Initialize(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		api:      enforcer,
		resource: enforcer,
	}
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "add api group policy failure",
			prepare: func() {
				enforcer.EXPECT().HasGroupingPolicy(gomock.Any(), gomock.Any()).Return(false)
				enforcer.EXPECT().AddGroupingPolicy(gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "add resource group policy failure",
			prepare: func() {
				enforcer.EXPECT().HasGroupingPolicy(gomock.Any(), gomock.Any()).Return(false).MaxTimes(5)
				enforcer.EXPECT().AddGroupingPolicy(gomock.Any(), gomock.Any()).Return(true, nil).MaxTimes(4)
				enforcer.EXPECT().AddGroupingPolicy(gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "add api policy failure",
			prepare: func() {
				enforcer.EXPECT().HasGroupingPolicy(gomock.Any(), gomock.Any()).Return(true).MaxTimes(8)
				enforcer.EXPECT().HasPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(false)
				enforcer.EXPECT().AddPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "initialize successfully",
			prepare: func() {
				enforcer.EXPECT().HasGroupingPolicy(gomock.Any(), gomock.Any()).Return(true).MaxTimes(8)
				enforcer.EXPECT().HasPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(false).AnyTimes()
				enforcer.EXPECT().AddPolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			err := srv.Initialize()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestAuthorizeService_CanAccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		api:    enforcer,
		logger: logger.GetLogger("Service", "AuthTest"),
	}
	t.Run("check failure", func(t *testing.T) {
		enforcer.EXPECT().Enforce("Admin", "AdminAccessResource", "write").Return(false, fmt.Errorf("err"))
		ok := srv.CanAccess(accesscontrol.RoleAdmin, accesscontrol.AdminAccessResource, accesscontrol.Write)
		assert.False(t, ok)
	})
	t.Run("can access", func(t *testing.T) {
		enforcer.EXPECT().Enforce("Admin", "AdminAccessResource", "write").Return(true, nil)
		ok := srv.CanAccess(accesscontrol.RoleAdmin, accesscontrol.AdminAccessResource, accesscontrol.Write)
		assert.True(t, ok)
	})
}

func TestAuthorizeService_CheckResourceACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		resource: enforcer,
		logger:   logger.GetLogger("Service", "AuthTest"),
	}
	t.Run("check failure", func(t *testing.T) {
		enforcer.EXPECT().Enforce("Admin", "123", "Component", "abc", "write").Return(false, fmt.Errorf("err"))
		ok := srv.CheckResourceACL(&modelpkg.ResourceACLParam{
			Role:     accesscontrol.RoleAdmin,
			OrgID:    123,
			Category: accesscontrol.Component,
			Resource: "abc",
			Action:   accesscontrol.Write,
		})
		assert.False(t, ok)
	})
	t.Run("can access", func(t *testing.T) {
		enforcer.EXPECT().Enforce("Admin", "123", "Component", "abc", "write").Return(true, nil)
		ok := srv.CheckResourceACL(&modelpkg.ResourceACLParam{
			Role:     accesscontrol.RoleAdmin,
			OrgID:    123,
			Category: accesscontrol.Component,
			Resource: "abc",
			Action:   accesscontrol.Write,
		})
		assert.True(t, ok)
	})
}

func TestAuthorizeService_CheckResourcesACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		resource: enforcer,
		logger:   logger.GetLogger("Service", "AuthTest"),
	}
	enforcer.EXPECT().BatchEnforce(gomock.Any()).Return(nil, fmt.Errorf("err"))
	ok, err := srv.CheckResourcesACL([]modelpkg.ResourceACLParam{{
		Role:     accesscontrol.RoleAdmin,
		OrgID:    123,
		Category: accesscontrol.Component,
		Resource: "abc",
		Action:   accesscontrol.Write,
	}})
	assert.Nil(t, ok)
	assert.Error(t, err)
}

func TestAuthorizeService_RemoveResourcesACL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		resource: enforcer,
		logger:   logger.GetLogger("Service", "AuthTest"),
	}
	enforcer.EXPECT().RemoveFilteredNamedPolicy("p", 1, "123", accesscontrol.Component.String()).Return(false, fmt.Errorf("err"))
	err := srv.RemoveResourcePoliciesByCategory(123, accesscontrol.Component)
	assert.Error(t, err)
}

func TestAuthorizeService_AddResourcePolicy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	enforcer := casbinmock.NewMockIEnforcer(ctrl)
	srv := &authorizeService{
		resource: enforcer,
		logger:   logger.GetLogger("Service", "AuthTest"),
	}
	t.Run("add failure", func(t *testing.T) {
		enforcer.EXPECT().HasPolicy("Admin", "123", "Component", "abc", "write").Return(false)
		enforcer.EXPECT().AddPolicy("Admin", "123", "Component", "abc", "write").Return(false, fmt.Errorf("err"))
		err := srv.AddResourcePolicy(&modelpkg.ResourceACLParam{
			Role:     accesscontrol.RoleAdmin,
			OrgID:    123,
			Category: accesscontrol.Component,
			Resource: "abc",
			Action:   accesscontrol.Write,
		})
		assert.Error(t, err)
	})
	t.Run("add ok", func(t *testing.T) {
		enforcer.EXPECT().HasPolicy("Admin", "123", "Component", "abc", "write").Return(false)
		enforcer.EXPECT().AddPolicy("Admin", "123", "Component", "abc", "write").Return(false, nil)
		err := srv.AddResourcePolicy(&modelpkg.ResourceACLParam{
			Role:     accesscontrol.RoleAdmin,
			OrgID:    123,
			Category: accesscontrol.Component,
			Resource: "abc",
			Action:   accesscontrol.Write,
		})
		assert.NoError(t, err)
	})
}
