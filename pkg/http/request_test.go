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
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/constant"
)

func TestRequest(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:80"+constant.LoginPage, http.NoBody)
	req.RequestURI = constant.LoginPage
	assert.NoError(t, err)
	c := &gin.Context{Request: req}
	assert.True(t, IsLoginPage(c))
	req.RequestURI = "/dashboard"
	assert.True(t, IsStaticResource(c))
	assert.False(t, IsLoginPage(c))
	req.RequestURI = constant.APIPathPrefix + "/dashboard"
	assert.True(t, IsAPIRequest(c))
	assert.False(t, IsStaticResource(c))
}
