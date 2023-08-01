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

// Tag represents term of tags.
type Tag struct {
	BaseModel

	OrgID int64  `json:"-" gorm:"column:org_id;index:u_idx_tags,unique"`
	Term  string `gorm:"column:term;index:u_idx_tags,unique"`
}

// ResourceTag represents resource's tags information.
type ResourceTag struct {
	BaseModel

	OrgID       int64        `json:"-" gorm:"column:org_id;index:u_idx_resource_tag,unique"`
	TagID       int64        `json:"-" gorm:"column:tag_id;index:u_idx_resource_tag,unique"`
	ResourceUID string       `json:"resourceUid" gorm:"column:resource_uid;index:u_idx_resource_tag,unique"`
	Type        ResourceType `json:"type" gorm:"column:type;index:u_idx_resource_tag,unique"`
}
