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

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight"
	"github.com/lindb/linsight/accesscontrol"
	modelpkg "github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

//go:generate mockgen -source=./authorize.go -destination=./authorize_mock.go -package=service

//go:generate mockgen -destination=./casbin/enforcer_mock.go -package=casbin github.com/casbin/casbin/v2 IEnforcer

// for testing
var (
	newAdapterByDBWithCustomTableFn = gormadapter.NewAdapterByDBWithCustomTable
	newModelFromStringFn            = model.NewModelFromString
	newEnforcerFn                   = casbin.NewEnforcer
)

// AuthorizeService represents user authorize service interface.
type AuthorizeService interface {
	// Initialize initializes access control policies.
	Initialize() error
	// CanAccess checks api if can access for user role.
	CanAccess(
		role accesscontrol.RoleType,
		resource accesscontrol.ResourceType,
		action accesscontrol.ActionType,
	) bool
	// AddResourcePolicy adds resource level acl policy.
	AddResourcePolicy(aclParam *modelpkg.ResourceACLParam) error
	// RemoveResourcePoliciesByCategory removes resource level acl policies by category.
	RemoveResourcePoliciesByCategory(orgID int64, category accesscontrol.ResourceCategory) error
	// UpdateResourceRole updates the acl role for resource.
	UpdateResourceRole(alcParam *modelpkg.ResourceACLParam) error
	// CheckResourceACL checks resource if can be accesed by given param.
	CheckResourceACL(aclParam *modelpkg.ResourceACLParam) bool
	// CheckResourcesACL checks resource list if can be accesed by given params.
	CheckResourcesACL(aclParams []modelpkg.ResourceACLParam) ([]bool, error)
}

// authorizeService implements AuthorizeService interface.
type authorizeService struct {
	db       dbpkg.DB
	api      casbin.IEnforcer
	resource casbin.IEnforcer

	logger logger.Logger
}

// NewAuthorizeService creates an AuthorizeService instance.
func NewAuthorizeService(db dbpkg.DB) AuthorizeService {
	return &authorizeService{
		db:       db,
		api:      createModel(linsight.RBACAPI, "api_rules", db),
		resource: createModel(linsight.ABACResource, "resource_rules", db),
		logger:   logger.GetLogger("Service", "Authorize"),
	}
}

// Initialize initializes access control policies.
func (srv *authorizeService) Initialize() error {
	// initialize roles
	if err := srv.addGroupingPolicy(srv.api); err != nil {
		return err
	}
	if err := srv.addGroupingPolicy(srv.resource); err != nil {
		return err
	}
	// initialize api acl policies
	policies := accesscontrol.BuildPolicyDefinitions()
	for _, policy := range policies {
		// add policy
		if hasPolicy := srv.api.HasPolicy(policy.RoleType.String(), policy.Resource.String(), policy.Action.String()); !hasPolicy {
			_, err := srv.api.AddPolicy(policy.RoleType.String(), policy.Resource.String(), policy.Action.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// CanAccess checks api if can access for user role.
func (srv *authorizeService) CanAccess(
	role accesscontrol.RoleType,
	resource accesscontrol.ResourceType,
	action accesscontrol.ActionType,
) bool {
	ok, err := srv.api.Enforce(role.String(), resource.String(), action.String())
	if err != nil {
		srv.logger.Error("check api acl failure", logger.Any("role", role),
			logger.Any("resource", resource), logger.Any("action", action), logger.Error(err))
		return false
	}
	return ok
}

// AddResourcePolicy adds resource level acl policy.
func (srv *authorizeService) AddResourcePolicy(aclParam *modelpkg.ResourceACLParam) error {
	params := aclParam.ToParams()
	// add policy
	if hasPolicy := srv.resource.HasPolicy(params...); !hasPolicy {
		_, err := srv.resource.AddPolicy(params...)
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveResourcePoliciesByCategory removes resource level acl policies by category.
func (srv *authorizeService) RemoveResourcePoliciesByCategory(orgID int64, category accesscontrol.ResourceCategory) error {
	_, err := srv.resource.RemoveFilteredNamedPolicy("p", 1, fmt.Sprintf("%d", orgID), category.String())
	return err
}

// UpdateResourceRole updates the acl role for resource.
func (srv *authorizeService) UpdateResourceRole(alcParam *modelpkg.ResourceACLParam) error {
	_, err := srv.resource.UpdateFilteredPolicies(
		[][]string{alcParam.ToStringParams()},
		1, fmt.Sprintf("%d", alcParam.OrgID), alcParam.Category.String(), alcParam.Resource, alcParam.Action.String())
	return err
}

// CheckResourceACL checks resource if can be accesed by given param.
func (srv *authorizeService) CheckResourceACL(aclParam *modelpkg.ResourceACLParam) bool {
	params := aclParam.ToParams()
	ok, err := srv.resource.Enforce(params...)
	if err != nil {
		srv.logger.Error("check resource acl failure", logger.Any("params", params), logger.Error(err))
		return false
	}
	return ok
}

// CheckResourcesACL checks resource list if can be accesed by given params.
func (srv *authorizeService) CheckResourcesACL(aclParams []modelpkg.ResourceACLParam) ([]bool, error) {
	var batch [][]any
	for _, p := range aclParams {
		batch = append(batch, (&p).ToParams())
	}
	return srv.resource.BatchEnforce(batch)
}

// addGroupingPolicy adds casbin grouping policies.
func (srv *authorizeService) addGroupingPolicy(enforcer casbin.IEnforcer) error {
	// initialize roles
	roles := accesscontrol.BuildRoleDefinitions()
	for _, role := range roles {
		if hasPolicy := enforcer.HasGroupingPolicy(role.RoleType.String(), role.Extends.String()); !hasPolicy {
			_, err := enforcer.AddGroupingPolicy(role.RoleType.String(), role.Extends.String())
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// createModel creates casbin policy model.
func createModel(modelStr, table string, db dbpkg.DB) casbin.IEnforcer {
	adapter, err := newAdapterByDBWithCustomTableFn(db.RawDB(), nil, table)
	if err != nil {
		panic(fmt.Sprintf("failed to create canbin grom adapter: %v", err))
	}
	m, err := newModelFromStringFn(modelStr)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin model: %v", err))
	}
	// load model configuration file and policy store adapter
	api, err := newEnforcerFn(m, adapter)
	if err != nil {
		panic(fmt.Sprintf("failed to create casbin enforcer: %v", err))
	}
	return api
}
