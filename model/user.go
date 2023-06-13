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

import "github.com/lindb/linsight/accesscontrol"

type User struct {
	BaseModel

	// current selected org.
	OrgID int64 `gorm:"column:org_id"`

	UID        string `gorm:"column:uid;u_idx_user_uid,unique"`
	Name       string `gorm:"column:name"`
	Email      string `gorm:"column:email;index:u_idx_user_email,unique"`
	Password   string `gorm:"column:password"`
	Salt       string `gorm:"column:salt"`
	IsDisabled bool   `gorm:"column:is_disabled"`
}

// Preference represents the preference of user.
type Preference struct {
	BaseModel

	OrgID    int64  `json:"-" gorm:"column:org_id;index:u_idx_pref_org,unique"`
	UserID   int64  `json:"-" gorm:"column:user_id;index:u_idx_pref_user,unique"`
	HomePage string `json:"homePage" gorm:"column:home_page"`
	Theme    string `json:"theme" gorm:"column:theme"`
	// use pointer to fix gorm not update bool value
	Collapsed *bool `json:"collapsed" gorm:"column:collapsed"`
}

type UserToken struct {
	BaseModel

	UserID int64 `json:"-" gorm:"column:user_id"`

	Token     string `json:"token" gorm:"column:token;index:u_idx_user_token,unique"`
	UserAgent string `json:"userAgent" gorm:"column:user_agent"`
	ClientIP  string `json:"clientIp" gorm:"column:client_ip"`
}

type OrgUser struct {
	BaseModel

	UserID int64                  `gorm:"column:user_id;index:u_idx_org_user,unique"`
	OrgID  int64                  `gorm:"column:org_id;index:u_idx_org_user,unique"`
	Role   accesscontrol.RoleType `gorm:"column:role"`
}

// LoginUser represents the parameters of login request.
type LoginUser struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type SignedUser struct {
	User       *User                  `json:"-"`
	Name       string                 `json:"name"`
	Email      string                 `json:"email"`
	IsDisabled bool                   `json:"isDisabled"`
	Org        *Org                   `json:"org"`
	Role       accesscontrol.RoleType `json:"role"`
	Preference *Preference            `json:"preference,omitempty"`
}
