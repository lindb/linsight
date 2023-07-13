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
	"github.com/tidwall/gjson"
	"gorm.io/datatypes"
)

// Chart represents chart basic information.
type Chart struct {
	BaseModel

	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_chart_org_title,unique"`

	UID         string `json:"uid" gorm:"column:uid;index:u_idx_chart_uid,unique"`
	Title       string `json:"title" gorm:"column:title;index:u_idx_chart_org_title,unique"`
	Desc        string `json:"description,omitempty" gorm:"column:desc"`
	Integration string `json:"integration" gorm:"column:integration"`
	Type        string `json:"type" gorm:"column:type"`
	Version     int    `json:"version,omitempty" gorm:"column:version"`

	Model datatypes.JSON `json:"model,omitempty" gorm:"column:model"`

	IsStarred bool `json:"isStarred,omitempty" gorm:"-"`
}

// ReadMeta reads metadata from config json.
func (c *Chart) ReadMeta() {
	json := c.Model.String()
	c.UID = gjson.Get(json, "uid").String()
	c.Title = gjson.Get(json, "title").String()
	c.Desc = gjson.Get(json, "description").String()
	c.Integration = gjson.Get(json, "integration").String()
	c.Type = gjson.Get(json, "type").String()
}

// SearchChartRequest represents search chart request params.
type SearchChartRequest struct {
	PagingParam
	Title     string    `form:"title" json:"title"`
	Ownership Ownership `form:"ownership" json:"ownership"`
	Tags      []string  `form:"tags" json:"tags"`
}

// ChartInfo represents chart basic information without config json.
type ChartInfo struct {
	UID         string         `json:"uid"`
	Title       string         `json:"title"`
	Desc        string         `json:"description,omitempty"`
	Type        string         `json:"type"`
	Integration string         `json:"integration,omitempty"`
	Dashboards  int            `json:"dashboards"`
	Model       datatypes.JSON `json:"model,omitempty"`
}
