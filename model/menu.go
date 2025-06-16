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

	"gorm.io/datatypes"

	"github.com/lindb/linsight/accesscontrol"
)

// Menus represents Menu slice.
type Menus []*Menu

// ToTree returns tree structure.
func (menus Menus) ToTree() Menus {
	// 1. cache all nodes
	menuMap := make(map[string]*Menu)
	for _, menu := range menus {
		menuMap[menu.UID] = menu
	}

	// 2. build tree
	var rootMenus Menus
	for _, menu := range menus {
		if menu.ParentUID == "" {
			rootMenus = append(rootMenus, menu)
		} else {
			parent, ok := menuMap[menu.ParentUID]
			if ok {
				parent.Children = append(parent.Children, menu)
			}
		}
	}

	sortFn := func(nodes Menus) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Order < nodes[j].Order
		})
	}

	// 3. sort children of all node
	sortFn(rootMenus)
	for _, menu := range rootMenus {
		sortFn(menu.Children)
	}

	return rootMenus
}

// Menu represents product definition menu.
type Menu struct {
	BaseModel

	UID string `json:"uid" gorm:"column:uid;index:u_idx_menu_uid,unique"`

	Label      string         `json:"label" gorm:"column:label"`
	Path       string         `json:"path,omitempty" gorm:"column:path"`
	Icon       string         `json:"icon,omitempty" gorm:"column:icon"`
	Page       string         `json:"page,omitempty" gorm:"column:page"`
	PageConfig datatypes.JSON `json:"pageConfig,omitempty" gorm:"column:page_config"`
	// default role
	Role  accesscontrol.RoleType `json:"role" gorm:"column:role"`
	Order int                    `json:"order" gorm:"column:order"`

	ParentUID string  `json:"parentUID,omitempty" gorm:"column:parent_uid"`
	Children  []*Menu `json:"children,omitempty" gorm:"-"`
}

// OrgMenu represents menu for organization.
type OrgMenu struct {
	BaseModel

	OrgID  int64 `json:"-" gorm:"column:org_id;index:u_idx_org_menu_org_menu_id,unique"`
	MenuID int64 `json:"-" gorm:"column:menu_id;index:u_idx_org_menu_org_menu_id,unique"`

	// role for org. level
	Role accesscontrol.RoleType `gorm:"column:role"`
}

// OrgMenuInfo represents menu basic info for organization.
type OrgMenuInfo struct {
	MenuUID string                 `json:"menuUid"`
	Role    accesscontrol.RoleType `json:"role" binding:"required"`
}
