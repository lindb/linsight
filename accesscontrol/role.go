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

package accesscontrol

type RoleType string

const (
	// Lin represents super admin.
	RoleLin       RoleType = "Lin"
	RoleAdmin     RoleType = "Admin"
	RoleEditor    RoleType = "Editor"
	RoleViewer    RoleType = "Viewer"
	RoleAnonymous RoleType = "Anonymous"
)

func (rt RoleType) String() string {
	return string(rt)
}

type Role struct {
	RoleType RoleType
	Extends  RoleType
}

type RoleBuilder struct {
	roles []Role
}

func NewRoleBuilder() *RoleBuilder {
	return &RoleBuilder{}
}

func (rb *RoleBuilder) AddRole(roleType, extends RoleType) *RoleBuilder {
	rb.roles = append(rb.roles, Role{
		RoleType: roleType,
		Extends:  extends,
	})
	return rb
}

func (rb *RoleBuilder) Build() []Role {
	return rb.roles
}

func BuildRoleDefinitions() []Role {
	return NewRoleBuilder().
		AddRole(RoleLin, RoleAdmin).
		AddRole(RoleAdmin, RoleEditor).
		AddRole(RoleEditor, RoleViewer).
		AddRole(RoleViewer, RoleAnonymous).
		Build()
}
