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
	"github.com/lindb/common/pkg/http"

	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// LoginAPI represents login related api handlers.
type LoginAPI struct {
	deps *depspkg.API
}

// NewLoginAPI creates a LoginAPI instance.
func NewLoginAPI(deps *depspkg.API) *LoginAPI {
	return &LoginAPI{
		deps: deps,
	}
}

// Login logins system via username/password.
func (api *LoginAPI) Login(c *gin.Context) {
	loginUser := &model.LoginUser{}
	if err := c.ShouldBind(loginUser); err != nil {
		http.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	// authenticate user info
	user, err := api.deps.AuthenticateSrv.Authenticate(ctx, loginUser)
	if err != nil {
		http.Error(c, err)
		return
	}
	// create token based signed in user info
	token, err := api.deps.AuthenticateSrv.CreateToken(ctx, user, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		http.Error(c, err)
		return
	}

	// FIXME: need modify/config
	cookieCfg := api.deps.Config.Cookie
	c.SetCookie(cookieCfg.Name, token.Token, int(cookieCfg.MaxAge.Duration().Seconds()), "/", "", false, true)
	http.OK(c, "Signed in!")
}

// Logout logouts system, removed login session.
func (api *LoginAPI) Logout(c *gin.Context) {
	cookieCfg := api.deps.Config.Cookie
	cookieName := cookieCfg.Name
	token, err := c.Cookie(cookieName)
	if err != nil {
		http.Error(c, err)
		return
	}
	c.SetCookie(cookieName, token, -1, "/", "", false, true)
	http.OK(c, "Logout!")
}
