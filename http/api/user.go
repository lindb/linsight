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

package api

import (
	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// UserAPI represents user related api handlers.
type UserAPI struct {
	deps *depspkg.API
}

// NewUserAPI creates an UserAPI instance.
func NewUserAPI(deps *depspkg.API) *UserAPI {
	return &UserAPI{
		deps: deps,
	}
}

// CreateUser creates a new user.
func (api *UserAPI) CreateUser(c *gin.Context) {
	user := &model.CreateUserRequest{}
	if err := c.ShouldBind(user); err != nil {
		httppkg.Error(c, err)
		return
	}

	uid, err := api.deps.UserSrv.CreateUser(c.Request.Context(), user)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateUser updates user basic information.
func (api *UserAPI) UpdateUser(c *gin.Context) {
	user := &model.User{}
	if err := c.ShouldBind(user); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.UserSrv.UpdateUser(c.Request.Context(), user); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, user)
}

// SearchUser searches users by given params.
func (api *UserAPI) SearchUser(c *gin.Context) {
	req := &model.SearchUserRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	users, total, err := api.deps.UserSrv.SearchUser(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total": total,
		"users": users,
	})
}

// GetUserByUID returns user by uid.
func (api *UserAPI) GetUserByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	user, err := api.deps.UserSrv.GetUserByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, user)
}

// DisableUserByUID disables user by uid.
func (api *UserAPI) DisableUserByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	err := api.deps.UserSrv.DisableUser(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "User disabled")
}

// EnableUserByUID enables user by uid.
func (api *UserAPI) EnableUserByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	err := api.deps.UserSrv.EnableUser(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "User enabled")
}

// GetPreference returns the preference of current signed user.
func (api *UserAPI) GetPreference(c *gin.Context) {
	pref, err := api.deps.UserSrv.GetPreference(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, pref)
}

// SavePreference saves the preference of current signed user.
func (api *UserAPI) SavePreference(c *gin.Context) {
	pref := &model.Preference{}
	if err := c.ShouldBind(pref); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.UserSrv.SavePreference(c.Request.Context(), pref); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Preferences saved")
}

// ChangePassword changes user password.
func (api *UserAPI) ChangePassword(c *gin.Context) {
	changePWD := &model.ChangeUserPassword{}
	if err := c.ShouldBind(changePWD); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.UserSrv.ChangePassword(c.Request.Context(), changePWD); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "User password changed")
}

// ResetPassword resets user password.
func (api *UserAPI) ResetPassword(c *gin.Context) {
	resetPWD := &model.ResetUserPassword{}
	if err := c.ShouldBind(resetPWD); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.UserSrv.ResetPassword(c.Request.Context(), resetPWD); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "User password changed")
}

// GetOrgListByUserUID returns the org list than user belong.
func (api *UserAPI) GetOrgListByUserUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	orgList, err := api.deps.UserSrv.GetOrgListByUserUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, orgList)
}

// AddOrg adds user org related(join org).
func (api *UserAPI) AddOrg(c *gin.Context) {
	userOrg := &model.UserOrgInfo{}
	if err := c.ShouldBind(userOrg); err != nil {
		httppkg.Error(c, err)
		return
	}
	userOrg.UserUID = c.Param(constant.UID)
	if err := api.deps.UserSrv.AddOrg(c.Request.Context(), userOrg); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Organization added")
}

// RemoveOrg removes user org related(leave org).
func (api *UserAPI) RemoveOrg(c *gin.Context) {
	userOrg := &model.UserOrgInfo{
		UserUID: c.Param(constant.UID),
		OrgUID:  c.Param("orgUid"),
	}
	if err := api.deps.UserSrv.RemoveOrg(c.Request.Context(), userOrg); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Organization removed")
}

// UpdateOrg updates user org realted.
func (api *UserAPI) UpdateOrg(c *gin.Context) {
	userOrg := &model.UserOrgInfo{}
	if err := c.ShouldBind(userOrg); err != nil {
		httppkg.Error(c, err)
		return
	}
	userOrg.UserUID = c.Param(constant.UID)
	if err := api.deps.UserSrv.UpdateOrg(c.Request.Context(), userOrg); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Organization updated")
}

// SwitchOrg switches target org.
func (api *UserAPI) SwitchOrg(c *gin.Context) {
	orgUID := c.Param("orgUid")
	if err := api.deps.UserSrv.SwitchOrg(c.Request.Context(), orgUID); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Organization removed")
}
