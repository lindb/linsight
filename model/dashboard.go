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
	"errors"
	"fmt"

	"github.com/lindb/common/pkg/encoding"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/constant"
)

type Ownership int
type JSONType datatypes.JSON

const (
	Any Ownership = iota
	Mine
	Shared
)

// Dashboard represents dasshboard basic information.
type Dashboard struct {
	BaseModel

	OrgID int64 `gorm:"column:org_id;index:u_idx_dashboard_org_title,unique"`

	UID     string `gorm:"column:uid;index:u_idx_dashboard_uid,unique"`
	Title   string `json:"title" gorm:"column:title;index:u_idx_dashboard_org_title,unique"`
	Desc    string `gorm:"column:desc"`
	Version int    `gorm:"column:version"`

	Config datatypes.JSON `gorm:"column:config"`

	// FIXME: need remove
	IsStarred bool `json:"isStarred" gorm:"-"`
}

// ReadMeta reads dashboard metadata from json data.
func (d *Dashboard) ReadMeta() error {
	var dashboardMap map[string]any
	if err := encoding.JSONUnmarshal(d.Config, &dashboardMap); err != nil {
		return err
	}
	title, ok := dashboardMap["title"]
	if !ok {
		return errors.New("title is required")
	}
	d.Title = fmt.Sprintf("%v", title)
	uid, ok := dashboardMap[constant.UID]
	if ok {
		d.UID = fmt.Sprintf("%v", uid)
	}
	return nil
}

// SearchDashboardRequest represents search dashboard request params.
type SearchDashboardRequest struct {
	PagingParam
	Title     string    `form:"title" json:"title"`
	Ownership Ownership `form:"ownership" json:"ownership"`
	Tags      []string  `form:"tags" json:"tags"`
}
