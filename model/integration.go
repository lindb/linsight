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

// Integration represents all integrations for observability of your application or infrastructure.
type Integration struct {
	BaseModel

	UID      string `json:"uid" gorm:"column:uid;index:u_idx_integrations_uid,unique"`
	Category string `json:"category" gorm:"column:category"`
	Title    string `json:"title" gorm:"column:title;index:u_idx_integrations_title,unique"`
	Desc     string `json:"desc" gorm:"column:desc"`
	Icon     string `json:"icon" gorm:"column:icon"`
	DocURL   string `json:"docUrl" gorm:"column:doc_url"`
}

// IntegrationConnection represents connection between integration and source(dashboard/chart etc.).
type IntegrationConnection struct {
	BaseModel

	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_ic,unique"`

	IntegrationUID string       `json:"integrationUID" gorm:"column:integration_uid;index:u_idx_ic,unique"`
	SourceUID      string       `json:"sourceUID" gorm:"column:source_uid;index:u_idx_ic,unique"`
	Type           ResourceType `json:"type" gorm:"column:type;index:u_idx_ic,unique"`
}
