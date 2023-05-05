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

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"

	"github.com/lindb/linsight"
	"github.com/lindb/linsight/accesscontrol"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

type AuthorizeService interface {
	Initialize() error
	CanAccess(
		role accesscontrol.RoleType,
		resource accesscontrol.ResourceType,
		action accesscontrol.ActionType,
	) bool
}

type authorizeService struct {
	db       dbpkg.DB
	enforcer casbin.IEnforcer
}

func NewAuthorizeService(db dbpkg.DB) AuthorizeService {
	adapter, err := gormadapter.NewAdapterByDB(db.RawDB())
	if err != nil {
		panic(fmt.Sprintf("failed to create canbin grom adapter: %v", err))
	}
	m, err := model.NewModelFromString(linsight.RBACModel)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin model: %v", err))
	}
	// Load model configuration file and policy store adapter
	enforcer, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin enforcer: %v", err))
	}
	return &authorizeService{
		db:       db,
		enforcer: enforcer,
	}
}

func (srv *authorizeService) Initialize() error {
	// initialize roles
	roles := accesscontrol.BuildRoleDefinitions()
	for _, role := range roles {
		if hasPolicy := srv.enforcer.HasGroupingPolicy(role.RoleType.String(), role.Extends.String()); !hasPolicy {
			_, err := srv.enforcer.AddGroupingPolicy(role.RoleType.String(), role.Extends.String())
			if err != nil {
				return err
			}
		}
	}
	// initialize policies
	policies := accesscontrol.BuildPolicyDefinitions()
	for _, policy := range policies {
		// add policy
		if hasPolicy := srv.enforcer.HasPolicy(policy.RoleType.String(), policy.Resource.String(), policy.Action.String()); !hasPolicy {
			_, err := srv.enforcer.AddPolicy(policy.RoleType.String(), policy.Resource.String(), policy.Action.String())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (srv *authorizeService) CanAccess(
	role accesscontrol.RoleType,
	resource accesscontrol.ResourceType,
	action accesscontrol.ActionType,
) bool {
	ok, err := srv.enforcer.Enforce(role.String(), resource.String(), action.String())
	if err != nil {
		// FIXME: add err log
		fmt.Println(err)
		return false
	}
	return ok
}
