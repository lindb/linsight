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

// Nav represents the navigation config information.
type Nav struct {
	BaseModel
	// current selected org.
	OrgID int64 `json:"-" gorm:"column:org_id;index:u_idx_nav_org_id,unique"`

	Config        datatypes.JSON `json:"config" gorm:"column:config"`
	DefaultConfig datatypes.JSON `json:"defaultConfig" gorm:"column:default_config"`
}
