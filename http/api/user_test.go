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

	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestUserAPI_GetPreference(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.GET("/user/preference", api.GetPreference)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get preference failure",
			prepare: func() {
				userSrv.EXPECT().GetPreference(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get preference successfully",
			prepare: func() {
				userSrv.EXPECT().GetPreference(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/user/preference", http.NoBody)
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

func TestUserAPI_SavePreference(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/user/preference", api.SavePreference)
	body := encoding.JSONMarshal(&model.Preference{})

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
			name: "update preference failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().SavePreference(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update preference successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().SavePreference(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/user/preference", tt.body)
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

func TestUserAPI_ChangePassword(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/user/password/change", api.ChangePassword)
	body := encoding.JSONMarshal(&model.ChangeUserPassword{
		OldPassword: "admin",
		NewPassword: "admin",
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
			name: "change password failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().ChangePassword(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				fmt.Println(resp)
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "change password successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().ChangePassword(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/user/password/change", tt.body)
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

func TestUserAPI_CreateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.POST("/users", api.CreateUser)
	body := encoding.JSONMarshal(&model.CreateUserRequest{
		UserName: "user",
		Password: "pwd",
		Email:    "email",
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
			name: "create user failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create user successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return("1234", nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Equal(t, `"1234"`, resp.Body.String())
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/users", tt.body)
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

func TestUserAPI_SearchUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/users", api.SearchUser)
	params := encoding.JSONMarshal(&model.SearchUserRequest{})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "cannot get params",
			body: bytes.NewBuffer([]byte("{abc")),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search user failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				userSrv.EXPECT().SearchUser(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search user successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				userSrv.EXPECT().SearchUser(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/users", tt.body)
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

func TestUserAPI_UpdateUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/users", api.UpdateUser)
	body := encoding.JSONMarshal(&model.User{})

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
			name: "update user failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update user successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/users", tt.body)
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

func TestUserAPI_GetUserByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.GET("/users/:uid", api.GetUserByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get user failure",
			prepare: func() {
				userSrv.EXPECT().GetUserByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get user successfully",
			prepare: func() {
				userSrv.EXPECT().GetUserByUID(gomock.Any(), "1234").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/users/1234", http.NoBody)
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

func TestUserAPI_DisableserByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/users/:uid/disable", api.DisableUserByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "disable user failure",
			prepare: func() {
				userSrv.EXPECT().DisableUser(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "disable user successfully",
			prepare: func() {
				userSrv.EXPECT().DisableUser(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/users/1234/disable", http.NoBody)
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

func TestUserAPI_EnableUserByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/users/:uid/enable", api.EnableUserByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "enable user failure",
			prepare: func() {
				userSrv.EXPECT().EnableUser(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "enable user successfully",
			prepare: func() {
				userSrv.EXPECT().EnableUser(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/users/1234/enable", http.NoBody)
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

func TestUserAPI_GetOrgListByUserUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.GET("/users/:uid/orgs", api.GetOrgListByUserUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get org list failure",
			prepare: func() {
				userSrv.EXPECT().GetOrgListByUserUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get org list successfully",
			prepare: func() {
				userSrv.EXPECT().GetOrgListByUserUID(gomock.Any(), "1234").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/users/1234/orgs", http.NoBody)
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

func TestUserAPI_AddOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.POST("/users/:uid/orgs", api.AddOrg)
	body := encoding.JSONMarshal(&model.UserOrgInfo{})

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
			name: "add user's org failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().AddOrg(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "add user's org successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().AddOrg(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/users/1234/orgs", tt.body)
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

func TestUserAPI_UpdateOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.PUT("/users/:uid/orgs", api.UpdateOrg)
	body := encoding.JSONMarshal(&model.UserOrgInfo{})

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
			name: "update user's org failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().UpdateOrg(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update user's org successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().UpdateOrg(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/users/1234/orgs", tt.body)
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

func TestUserAPI_DeleteOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userSrv := service.NewMockUserService(ctrl)
	r := gin.New()
	api := NewUserAPI(&deps.API{
		UserSrv: userSrv,
	})
	r.DELETE("/users/:uid/orgs/:orgUid", api.RemoveOrg)
	body := encoding.JSONMarshal(&model.UserOrgInfo{})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "remove user's org failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().RemoveOrg(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "remove user's org successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				userSrv.EXPECT().RemoveOrg(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/users/1234/orgs/4321", tt.body)
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
