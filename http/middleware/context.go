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

package middleware

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
)

// InitContext initializes the signed context.
func InitContext(deps *depspkg.API) gin.HandlerFunc {
	return func(c *gin.Context) {
		initSignedUser(c, deps)

		c.Next()
	}
}

func initSignedUser(c *gin.Context, deps *depspkg.API) {
	token, err := c.Cookie(deps.Config.Cookie.Name)
	if err != nil {
		fmt.Println(err)
		return
	}
	signedUser := deps.UserCacheSrv.GetSignedUser(c.Request.Context(), token)
	fmt.Println(signedUser)
	if signedUser == nil {
		// evict cache after  user information changed
		userToken, err := deps.AuthenticateSrv.LookupToken(c.Request.Context(), token)
		if err != nil {
			// FIXME: add log
			fmt.Println(err)
			return
		}
		if userToken == nil {
			return
		}
		signedUser, err = deps.UserSrv.GetSignedUser(c.Request.Context(), userToken.UserID)
		if err != nil {
			// FIXME: add log
			fmt.Println(err)
			return
		}
		if signedUser != nil {
			// create token based signed in user info
			userToken, err = deps.AuthenticateSrv.CreateToken(c.Request.Context(), signedUser.User, c.ClientIP(), c.Request.UserAgent())
			if err != nil {
				fmt.Println(err)
				return
			}

			newToken := userToken.Token
			// FIXME: need modify/config
			cookieCfg := deps.Config.Cookie
			signedUser.Token = newToken // set token
			c.SetCookie(cookieCfg.Name, newToken, int(cookieCfg.MaxAge.Duration().Seconds()), "/", "", false, true)
			// cache signed user using new token
			deps.UserCacheSrv.StoreSignedUser(c.Request.Context(), newToken, signedUser)
		}
	}
	if signedUser == nil {
		return
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), constant.LinSightSignedKey, signedUser))
}
