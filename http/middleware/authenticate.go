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
	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/pkg/http"
	"github.com/lindb/linsight/pkg/util"
)

func Authenticate(deps *depspkg.API) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := util.GetUser(c.Request.Context())
		if user != nil {
			c.Next()
			return
		}

		if http.IsAPIRequest(c) {
			httppkg.Forbidden(c)
		} else {
			httppkg.Redirect(c, constant.LoginPage)
		}
		c.Abort()
	}
}
