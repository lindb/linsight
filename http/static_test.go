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
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
)

func TestStaticHandler_Panic(t *testing.T) {
	defer func() {
		fsSubFn = fs.Sub
	}()
	assert.Panics(t, func() {
		fsSubFn = func(_ fs.FS, _ string) (fs.FS, error) {
			return nil, fmt.Errorf("err")
		}
		handleStatic(nil)
		assert.True(t, false)
	})
}

func TestStaticHandler(t *testing.T) {
	r := gin.New()
	r.GET("/api/ok", func(c *gin.Context) {
		httppkg.OK(c, "ok")
	})

	handleStatic(r)

	cases := []struct {
		name   string
		ctx    context.Context
		path   string
		assert func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "home page",
			path: "/",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "ok page",
			path: "/api/ok",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "no route",
			path: "/api/404",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusNotFound, resp.Code)
			},
		},
		{
			name: "forward home",
			path: "/css",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "not login",
			path: constant.LoginPage,
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
		{
			name: "login forward homepage",
			path: constant.LoginPage,
			ctx:  context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{}),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusFound, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			ctx := tt.ctx
			if ctx == nil {
				ctx = context.TODO()
			}
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, tt.path, http.NoBody)
			req.RequestURI = tt.path
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}
