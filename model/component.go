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

package model

import (
	"sort"

	"github.com/lindb/linsight/accesscontrol"
)

// Components represents Component slice.
type Components []*Component

// ToTree returns tree structure.
func (cmps Components) ToTree() Components {
	// 1. cache all nodes
	cmpMap := make(map[string]*Component)
	for _, cmp := range cmps {
		cmpMap[cmp.UID] = cmp
	}

	// 2. build tree
	var rootCmps Components
	for _, cmp := range cmps {
		if cmp.ParentUID == "" {
			rootCmps = append(rootCmps, cmp)
		} else {
			parent, ok := cmpMap[cmp.ParentUID]
			if ok {
				parent.Children = append(parent.Children, cmp)
			}
		}
	}

	sortFn := func(nodes Components) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Order < nodes[j].Order
		})
	}

	// 3. sort children of all node
	sortFn(rootCmps)
	for _, cmp := range rootCmps {
		sortFn(cmp.Children)
	}

	return rootCmps
}

// Component represents product component.
type Component struct {
	BaseModel

	UID string `json:"uid" gorm:"column:uid;index:u_idx_cmp_uid,unique"`

	Label     string `json:"label" gorm:"column:label"`
	Path      string `json:"path,omitempty" gorm:"column:path"`
	Icon      string `json:"icon,omitempty" gorm:"column:icon"`
	Component string `json:"component,omitempty" gorm:"column:component"`
	// default role
	Role  accesscontrol.RoleType `json:"role" gorm:"column:role"`
	Order int                    `json:"order" gorm:"column:order"`

	ParentUID string       `json:"parentUID,omitempty" gorm:"column:parent_uid"`
	Children  []*Component `json:"children,omitempty" gorm:"-"`
}

// OrgComponent represents component item for organization.
type OrgComponent struct {
	BaseModel

	OrgID       int64 `json:"-" gorm:"column:org_id;index:u_idx_org_cmp_org_cmp_id,unique"`
	ComponentID int64 `json:"-" gorm:"column:component_id;index:u_idx_org_cmp_org_cmp_id,unique"`

	// role for org. level
	Role accesscontrol.RoleType `gorm:"column:role"`
}

// OrgComponentInfo represents component basic info for organization.
type OrgComponentInfo struct {
	ComponentUID string                 `json:"componentUid"`
	Role         accesscontrol.RoleType `json:"role" binding:"required"`
}
