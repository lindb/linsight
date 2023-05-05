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

package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/constant"
)

// IsAPIRequest returns if api request.
func IsAPIRequest(c *gin.Context) bool {
	return strings.HasPrefix(c.Request.RequestURI, constant.APIPathPrefix)
}

// IsStaticResource returns if static resource.
func IsStaticResource(c *gin.Context) bool {
	return c.Request.Method == http.MethodGet && !IsAPIRequest(c)
}

// IsLoginPage returns if login page.
func IsLoginPage(c *gin.Context) bool {
	return c.Request.RequestURI == constant.LoginPage
}
