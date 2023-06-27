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

func TestComponentAPI_LoadComponentTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmpSrv := service.NewMockComponentService(ctrl)
	r := gin.New()
	api := NewComponentAPI(&deps.API{
		CmpSrv: cmpSrv,
	})
	r.GET("/components", api.LoadComponentTree)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "load cmp tree failure",
			prepare: func() {
				cmpSrv.EXPECT().LoadComponentTree(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "load cmp tree successfully",
			prepare: func() {
				cmpSrv.EXPECT().LoadComponentTree(gomock.Any()).Return(model.Components{}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/components", http.NoBody)
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

func TestComponentAPI_CreateComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmpSrv := service.NewMockComponentService(ctrl)
	r := gin.New()
	api := NewComponentAPI(&deps.API{
		CmpSrv: cmpSrv,
	})
	r.POST("/components", api.CreateComponent)
	body := encoding.JSONMarshal(&model.Component{})

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
			name: "create cmp failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().CreateComponent(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create cmp successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().CreateComponent(gomock.Any(), gomock.Any()).Return("1234", nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/components", tt.body)
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

func TestComponentAPI_UpdateComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmpSrv := service.NewMockComponentService(ctrl)
	r := gin.New()
	api := NewComponentAPI(&deps.API{
		CmpSrv: cmpSrv,
	})
	r.PUT("/components", api.UpdateComponent)
	body := encoding.JSONMarshal(&model.Component{})

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
			name: "update cmp failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().UpdateComponent(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update cmp successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().UpdateComponent(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/components", tt.body)
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

func TestComponentAPI_SortComponents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmpSrv := service.NewMockComponentService(ctrl)
	r := gin.New()
	api := NewComponentAPI(&deps.API{
		CmpSrv: cmpSrv,
	})
	r.PUT("/components/sort", api.SortComponents)
	body := encoding.JSONMarshal(&model.Components{})

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
			name: "sort cmp failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().SortComponents(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "sort cmp successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				cmpSrv.EXPECT().SortComponents(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/components/sort", tt.body)
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

func TestComponentAPI_DeleteComponentByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cmpSrv := service.NewMockComponentService(ctrl)
	r := gin.New()
	api := NewComponentAPI(&deps.API{
		CmpSrv: cmpSrv,
	})
	r.DELETE("/components/:uid", api.DeleteComponentByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete cmp failure",
			prepare: func() {
				cmpSrv.EXPECT().DeleteComponentByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete cmp successfully",
			prepare: func() {
				cmpSrv.EXPECT().DeleteComponentByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/components/1234", http.NoBody)
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
