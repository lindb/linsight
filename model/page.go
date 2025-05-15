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

// Pages represents Page slice.
type Pages []*Page

// ToTree returns tree structure.
func (pages Pages) ToTree() Pages {
	// 1. cache all nodes
	pageMap := make(map[string]*Page)
	for _, page := range pages {
		pageMap[page.UID] = page
	}

	// 2. build tree
	var rootPages Pages
	for _, page := range pages {
		if page.ParentUID == "" {
			rootPages = append(rootPages, page)
		} else {
			parent, ok := pageMap[page.ParentUID]
			if ok {
				parent.Children = append(parent.Children, page)
			}
		}
	}

	sortFn := func(nodes Pages) {
		sort.Slice(nodes, func(i, j int) bool {
			return nodes[i].Order < nodes[j].Order
		})
	}

	// 3. sort children of all node
	sortFn(rootPages)
	for _, page := range rootPages {
		sortFn(page.Children)
	}

	return rootPages
}

// Page represents product definition page.
type Page struct {
	BaseModel

	UID string `json:"uid" gorm:"column:uid;index:u_idx_page_uid,unique"`

	Label      string         `json:"label" gorm:"column:label"`
	Path       string         `json:"path,omitempty" gorm:"column:path"`
	Icon       string         `json:"icon,omitempty" gorm:"column:icon"`
	Page       string         `json:"page,omitempty" gorm:"column:page"`
	PageConfig datatypes.JSON `json:"pageConfig,omitempty" gorm:"column:page_config"`
	// default role
	Role  accesscontrol.RoleType `json:"role" gorm:"column:role"`
	Order int                    `json:"order" gorm:"column:order"`

	ParentUID string  `json:"parentUID,omitempty" gorm:"column:parent_uid"`
	Children  []*Page `json:"children,omitempty" gorm:"-"`
}

// OrgPage represents page for organization.
type OrgPage struct {
	BaseModel

	OrgID  int64 `json:"-" gorm:"column:org_id;index:u_idx_org_page_org_page_id,unique"`
	PageID int64 `json:"-" gorm:"column:page_id;index:u_idx_org_page_org_page_id,unique"`

	// role for org. level
	Role accesscontrol.RoleType `gorm:"column:role"`
}

// OrgPageInfo represents page basic info for organization.
type OrgPageInfo struct {
	PageUID string                 `json:"pageUid"`
	Role    accesscontrol.RoleType `json:"role" binding:"required"`
}
