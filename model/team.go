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

// Team represents the team inforamtion.
type Team struct {
	BaseModel

	// current selected org.
	OrgID int64 `json:"-" gorm:"column:org_id"`

	UID   string `json:"uid" gorm:"column:uid;index:u_idx_team_uid,unique"`
	Name  string `json:"name" gorm:"column:name;index:u_idx_team_name,unique"`
	Email string `json:"email" gorm:"column:email"`
}

// SearchTeamRequest represents search team request params.
type SearchTeamRequest struct {
	PagingParam
	Name string `form:"name" json:"name"`
}
