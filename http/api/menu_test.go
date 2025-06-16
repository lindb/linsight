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
	"github.com/lindb/common/pkg/encoding"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/accesscontrol"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestMenuAPI_LoadMenuTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.GET("/menus", api.LoadMenuTree)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "load menu tree failure",
			prepare: func() {
				menuSrv.EXPECT().LoadMenuTree(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "load menu tree successfully",
			prepare: func() {
				menuSrv.EXPECT().LoadMenuTree(gomock.Any()).Return(model.Menus{}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/menus", http.NoBody)
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

func TestMenuAPI_CreateMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.POST("/menus", api.CreateMenu)
	body := encoding.JSONMarshal(&model.Menu{})

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
			name: "create menu failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().CreateMenu(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create menu successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().CreateMenu(gomock.Any(), gomock.Any()).Return("1234", nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/menus", tt.body)
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

func TestMenuAPI_UpdateMenu(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.PUT("/menus", api.UpdateMenu)
	body := encoding.JSONMarshal(&model.Menu{})

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
			name: "update menu failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().UpdateMenu(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update menu successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().UpdateMenu(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/menus", tt.body)
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

func TestMenuAPI_SortMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.PUT("/menus/sort", api.SortMenus)
	body := encoding.JSONMarshal(&model.Menus{})

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
			name: "sort menus failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().SortMenus(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "sort menus successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().SortMenus(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/menus/sort", tt.body)
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

func TestMenuAPI_DeleteMenuByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.DELETE("/menus/:uid", api.DeleteMenuByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete menu failure",
			prepare: func() {
				menuSrv.EXPECT().DeleteMenuByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete menu successfully",
			prepare: func() {
				menuSrv.EXPECT().DeleteMenuByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/menus/1234", http.NoBody)
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

func TestMenuAPI_SaveOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.PUT("/menus/:uid", api.SaveOrgMenus)
	body := encoding.JSONMarshal([]model.OrgMenuInfo{{Role: accesscontrol.RoleAdmin}})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name:    "bind params failure",
			body:    bytes.NewBuffer([]byte("xx")),
			prepare: func() {},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "save org menu failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().SaveOrgMenus(gomock.Any(), "1234", gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "save org menu successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().SaveOrgMenus(gomock.Any(), "1234", gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/menus/1234", tt.body)
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

func TestMenuAPI_UpdateRolesOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.PUT("/menus", api.UpdateRolesOfOrgMenus)
	body := encoding.JSONMarshal([]model.OrgMenuInfo{{Role: accesscontrol.RoleAdmin}})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name:    "bind params failure",
			body:    bytes.NewBuffer([]byte("xx")),
			prepare: func() {},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update role for org menus failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().UpdateRolesOfOrgMenus(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update role for org menus successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				menuSrv.EXPECT().UpdateRolesOfOrgMenus(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/menus", tt.body)
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

func TestMenuAPI_GetMenuTreeByCurrentOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.GET("/menus", api.GetMenuTreeByCurrentOrg)
	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get org menu failure",
			prepare: func() {
				menuSrv.EXPECT().GetMenuTreeByCurrentOrg(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get org menu successfully",
			prepare: func() {
				menuSrv.EXPECT().GetMenuTreeByCurrentOrg(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/menus", http.NoBody)
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

func TestMenuAPI_GetOrgMenus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	menuSrv := service.NewMockMenuService(ctrl)
	r := gin.New()
	api := NewMenuAPI(&deps.API{
		MenuSrv: menuSrv,
	})
	r.GET("/menus/:uid", api.GetOrgMenus)
	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get org menus failure",
			prepare: func() {
				menuSrv.EXPECT().GetOrgMenus(gomock.Any(), "123").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get org menus successfully",
			prepare: func() {
				menuSrv.EXPECT().GetOrgMenus(gomock.Any(), "123").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/menus/123", http.NoBody)
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
