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

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"

	"github.com/gin-gonic/gin"
)

func getUserToken(c *gin.Context, deps *depspkg.API) (*model.UserToken, error) {
	token, err := c.Cookie(deps.Config.Cookie.Name)
	if err != nil {
		return nil, err
	}
	return deps.AuthenticateSrv.LookupToken(c.Request.Context(), token)
}

func InitContext(deps *depspkg.API) gin.HandlerFunc {
	return func(c *gin.Context) {
		initSignedUser(c, deps)

		c.Next()
	}
}

func initSignedUser(c *gin.Context, deps *depspkg.API) {
	userToken, err := getUserToken(c, deps)
	if err != nil {
		// FIXME: add log
		fmt.Println(err)
		return
	}
	if userToken == nil {
		return
	}
	signedUser, err := deps.UserSrv.GetSignedUser(c.Request.Context(), userToken.UserID)
	if err != nil {
		// FIXME: add log
		fmt.Println(err)
		return
	}
	if signedUser == nil {
		return
	}
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), constant.LinSightSignedKey, signedUser))
}
