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

type ActionType string
type ResourceType string
type ResourceCategory string

const (
	Dashboard ResourceCategory = "Dashboard"
	Component ResourceCategory = "Component"
)

const (
	Read  ActionType = "read"
	Write ActionType = "write"
)

const (
	LinAccessResource    ResourceType = "LinAccessResource"
	AdminAccessResource  ResourceType = "AdminAccessResource"
	EditorAccessResource ResourceType = "EditorAccessResource"
	ViewerAccessResource ResourceType = "ViewerAccessResource"
)

func (c ResourceCategory) String() string {
	return string(c)
}

func (r ResourceType) String() string {
	return string(r)
}

func (a ActionType) String() string {
	return string(a)
}

type Policy struct {
	RoleType RoleType
	Resource ResourceType
	Action   ActionType
}

type PolicyBuilder struct {
	policies []Policy
}

func NewPlicyBuilder() *PolicyBuilder {
	return &PolicyBuilder{}
}

func (pb *PolicyBuilder) AddPolicy(role RoleType, resource ResourceType, action ActionType) *PolicyBuilder {
	pb.policies = append(pb.policies, Policy{
		RoleType: role,
		Resource: resource,
		Action:   action,
	})
	return pb
}

func (pb *PolicyBuilder) Build() []Policy {
	return pb.policies
}

func BuildPolicyDefinitions() []Policy {
	return NewPlicyBuilder().
		AddPolicy(RoleLin, LinAccessResource, Write).
		AddPolicy(RoleLin, LinAccessResource, Read).
		AddPolicy(RoleAdmin, AdminAccessResource, Write).
		AddPolicy(RoleAdmin, AdminAccessResource, Read).
		AddPolicy(RoleEditor, EditorAccessResource, Write).
		AddPolicy(RoleEditor, EditorAccessResource, Read).
		AddPolicy(RoleViewer, ViewerAccessResource, Write).
		AddPolicy(RoleViewer, ViewerAccessResource, Read).
		Build()
}
