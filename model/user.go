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

var collapsed = true

// DefaultUserPreference represents default preference if user not set.
var DefaultUserPreference = Preference{
	Collapsed: &collapsed,
	Theme:     "light",
}

// User represents the user basic information.
type User struct {
	BaseModel

	// current selected org.
	OrgID int64 `json:"-" gorm:"column:org_id"`

	UID        string `json:"uid" gorm:"column:uid;index:u_idx_user_uid,unique"`
	Name       string `json:"name" gorm:"column:name"`
	UserName   string `json:"userName" gorm:"column:user_name;index:u_idx_user_uname,unique"`
	Email      string `json:"email" gorm:"column:email;index:u_idx_user_email,unique"`
	Password   string `json:"-" gorm:"column:password"`
	Salt       string `json:"-" gorm:"column:salt"`
	IsDisabled *bool  `json:"isDisabled" gorm:"column:is_disabled"`
}

// CreateUserRequest represents create new user request
type CreateUserRequest struct {
	Name     string `json:"name"`
	UserName string `json:"userName" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Preference represents the preference of user.
type Preference struct {
	BaseModel

	UserID   int64  `json:"-" gorm:"column:user_id;index:u_idx_pref_user,unique"`
	HomePage string `json:"homePage" gorm:"column:home_page"`
	Theme    string `json:"theme" gorm:"column:theme"`
	// use pointer to fix gorm not update bool value
	Collapsed *bool `json:"collapsed" gorm:"column:collapsed"`
}

// UserToken represents user's login token and history.
type UserToken struct {
	BaseModel

	UserID int64 `json:"-" gorm:"column:user_id"`

	Token     string `json:"token" gorm:"column:token;index:u_idx_user_token,unique"`
	UserAgent string `json:"userAgent" gorm:"column:user_agent"`
	ClientIP  string `json:"clientIp" gorm:"column:client_ip"`
}

// OrgUser represents user and org related information.
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

// SignedUser represents the signed user information.
type SignedUser struct {
	User       *User                  `json:"-"`
	UserName   string                 `json:"userName"`
	Name       string                 `json:"name"`
	Email      string                 `json:"email"`
	IsDisabled bool                   `json:"isDisabled"`
	Org        *Org                   `json:"org"`
	Role       accesscontrol.RoleType `json:"role"`
	Preference *Preference            `json:"preference,omitempty"`
}

// ChangeUserPassword represents change user password params.
type ChangeUserPassword struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// SearchUserRequest represents search user request params.
type SearchUserRequest struct {
	PagingParam

	Query string `form:"query" json:"query"`
}
