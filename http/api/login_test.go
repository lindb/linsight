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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/config"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func TestLoginAPI_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	authSrv := service.NewMockAuthenticateService(ctrl)
	r := gin.New()
	cfg := config.NewDefaultServer()
	api := NewLoginAPI(&deps.API{
		Config:          cfg,
		AuthenticateSrv: authSrv,
	})
	r.POST("/login", api.Login)
	body := encoding.JSONMarshal(&model.LoginUser{
		Username: "user",
		Password: "pwd",
	})
	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "cannot get params",
			body: http.NoBody,
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "auth failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				authSrv.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create token failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				authSrv.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				authSrv.EXPECT().CreateToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				authSrv.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(&model.User{}, nil)
				authSrv.EXPECT().CreateToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&model.UserToken{}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				r := resp.Result()
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Len(t, r.Cookies(), 1)
				_ = r.Body.Close()
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/login", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			if tt.prepare != nil {
				tt.prepare()
			}
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestLoginAPI_Logout(t *testing.T) {
	r := gin.New()
	cfg := config.NewDefaultServer()
	api := NewLoginAPI(&deps.API{
		Config: cfg,
	})
	r.GET("/logout", api.Logout)

	cases := []struct {
		name    string
		prepare func(req *http.Request)
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name:    "cannot get cookie",
			prepare: func(req *http.Request) {},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "logout successfully",
			prepare: func(req *http.Request) {
				req.AddCookie(&http.Cookie{
					Name:  cfg.Cookie.Name,
					Value: "token",
				})
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/logout", http.NoBody)
			resp := httptest.NewRecorder()
			tt.prepare(req)
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}
