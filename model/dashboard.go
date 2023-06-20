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
	"encoding/json"
	"strings"

	"github.com/lindb/common/pkg/encoding"
	"github.com/tidwall/gjson"
	"gorm.io/datatypes"
)

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
)

type PermissionType int

const (
	PermissionUnknown PermissionType = iota
	PermissionMember
	PermissionAdmin
)

var permissionTypes = [...]string{"Unknown", "Member", "Admin"}

// String returns the string value of permission type.
func (p PermissionType) String() string {
	if p < PermissionUnknown || p > PermissionAdmin {
		return "Unknown"
	}
	return permissionTypes[p]
}

func (p PermissionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PermissionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := jsonUnmarshalFn(data, &s); err != nil {
		return err
	}
	*p = 0
	for i := 0; i < len(permissionTypes); i++ {
		if strings.Contains(s, permissionTypes[i]) {
			*p = PermissionType(i)
			return nil
		}
	}
	return nil
}

type Ownership int

const (
	Any Ownership = iota
	Mine
	Shared
)

// Dashboard represents dasshboard basic information.
type Dashboard struct {
	BaseModel

	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_dashboard_org_title,unique"`

	UID     string `json:"uid" gorm:"column:uid;index:u_idx_dashboard_uid,unique"`
	Title   string `json:"title" gorm:"column:title;index:u_idx_dashboard_org_title,unique"`
	Desc    string `json:"desc" gorm:"column:desc"`
	Version int    `gorm:"column:version"`

	Config datatypes.JSON `json:"-" gorm:"column:config"`

	// FIXME: need remove
	IsStarred bool `json:"isStarred" gorm:"-"`
}

// ReadMeta reads dashboard metadata from json data.
func (d *Dashboard) ReadMeta() {
	jsonData := d.Config.String()
	d.Title = gjson.Get(jsonData, "title").String()
	d.Desc = gjson.Get(jsonData, "description").String()
	d.UID = gjson.Get(jsonData, "uid").String()
}

// SearchDashboardRequest represents search dashboard request params.
type SearchDashboardRequest struct {
	PagingParam
	Title     string    `form:"title" json:"title"`
	Ownership Ownership `form:"ownership" json:"ownership"`
	Tags      []string  `form:"tags" json:"tags"`
}
