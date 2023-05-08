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

import "gorm.io/datatypes"

// Chart represents chart basic information.
type Chart struct {
	BaseModel

	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_chart_org_title,unique"`

	UID     string `json:"uid" gorm:"column:uid;index:u_idx_chart_uid,unique"`
	Title   string `json:"title" gorm:"column:title;index:u_idx_chart_org_title,unique"`
	Desc    string `json:"desc" gorm:"column:desc"`
	Version int    `json:"version" gorm:"column:version"`

	Config datatypes.JSON `json:"config" gorm:"column:config"`

	IsStarred bool `json:"isStarred" gorm:"-"`
}

// SearchChartRequest represents search chart request params.
type SearchChartRequest struct {
	PagingParam
	Title     string    `form:"title" json:"title"`
	Ownership Ownership `form:"ownership" json:"ownership"`
	Tags      []string  `form:"tags" json:"tags"`
}
