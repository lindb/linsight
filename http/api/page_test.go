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

func TestPageAPI_LoadPageTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.GET("/pages", api.LoadPageTree)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "load page tree failure",
			prepare: func() {
				pageSrv.EXPECT().LoadPageTree(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "load page tree successfully",
			prepare: func() {
				pageSrv.EXPECT().LoadPageTree(gomock.Any()).Return(model.Pages{}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/pages", http.NoBody)
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

func TestPageAPI_CreatePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.POST("/pages", api.CreatePage)
	body := encoding.JSONMarshal(&model.Page{})

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
			name: "create page failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().CreatePage(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create page successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().CreatePage(gomock.Any(), gomock.Any()).Return("1234", nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/pages", tt.body)
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

func TestPageAPI_UpdatePage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.PUT("/pages", api.UpdatePage)
	body := encoding.JSONMarshal(&model.Page{})

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
			name: "update page failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().UpdatePage(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update page successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().UpdatePage(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/pages", tt.body)
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

func TestPageAPI_SortPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.PUT("/pages/sort", api.SortPages)
	body := encoding.JSONMarshal(&model.Pages{})

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
			name: "sort pages failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().SortPages(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "sort pages successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().SortPages(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/pages/sort", tt.body)
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

func TestPageAPI_DeletePageByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.DELETE("/pages/:uid", api.DeletePageByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete page failure",
			prepare: func() {
				pageSrv.EXPECT().DeletePageByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete page successfully",
			prepare: func() {
				pageSrv.EXPECT().DeletePageByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/pages/1234", http.NoBody)
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

func TestPageAPI_SaveOrgPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.PUT("/pages/:uid", api.SaveOrgPages)
	body := encoding.JSONMarshal([]model.OrgPageInfo{{Role: accesscontrol.RoleAdmin}})

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
			name: "save org page failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().SaveOrgPages(gomock.Any(), "1234", gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "save org page successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().SaveOrgPages(gomock.Any(), "1234", gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/pages/1234", tt.body)
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

func TestPageAPI_UpdateRolesOrgPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.PUT("/pages", api.UpdateRolesOfOrgPages)
	body := encoding.JSONMarshal([]model.OrgPageInfo{{Role: accesscontrol.RoleAdmin}})

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
			name: "update role for org pages failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().UpdateRolesOfOrgPages(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update role for org pages successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				pageSrv.EXPECT().UpdateRolesOfOrgPages(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/pages", tt.body)
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

func TestPageAPI_GetPageTreeByCurrentOrg(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.GET("/pages", api.GetPageTreeByCurrentOrg)
	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get org page failure",
			prepare: func() {
				pageSrv.EXPECT().GetPageTreeByCurrentOrg(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get org page successfully",
			prepare: func() {
				pageSrv.EXPECT().GetPageTreeByCurrentOrg(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/pages", http.NoBody)
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

func TestPageAPI_GetOrgPages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pageSrv := service.NewMockPageService(ctrl)
	r := gin.New()
	api := NewPageAPI(&deps.API{
		PageSrv: pageSrv,
	})
	r.GET("/pages/:uid", api.GetOrgPages)
	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get org pages failure",
			prepare: func() {
				pageSrv.EXPECT().GetOrgPages(gomock.Any(), "123").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get org pages successfully",
			prepare: func() {
				pageSrv.EXPECT().GetOrgPages(gomock.Any(), "123").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/pages/123", http.NoBody)
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
