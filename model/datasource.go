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

// DatasourceType represents the type of datasource.
type DatasourceType = string

// Define all support datasource types.
var (
	LinDBDatasource DatasourceType = "lindb"
	LinGoDatasource DatasourceType = "lingo"
)

// Datasource represents datasource information.
type Datasource struct {
	BaseModel

	// ord id + name => unique key
	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_datasource_org_name,unique"`

	UID       string         `json:"uid" gorm:"column:uid;index:u_idx_datasource_uid,unique"`
	Name      string         `json:"name" gorm:"column:name;index:u_idx_datasource_org_name,unique"`
	Type      DatasourceType `json:"type" gorm:"column:type"`
	URL       string         `json:"url" gorm:"column:url"`
	TimeZone  string         `json:"timeZone" gorm:"column:time_zone"`
	Config    datatypes.JSON `json:"config" gorm:"column:config"`
	IsDefault bool           `json:"isDefault" gorm:"column:is_default"`
}
