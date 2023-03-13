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

func TestDatasourceAPI_CreateDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	r := gin.New()
	api := NewDatasourceAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
	})
	r.POST("/datasource", api.CreateDatasource)
	body := encoding.JSONMarshal(&model.Datasource{})

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
			name: "create data source failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				datasourceSrv.EXPECT().CreateDatasource(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create data source successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				datasourceSrv.EXPECT().CreateDatasource(gomock.Any(), gomock.Any()).Return("1234", nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/datasource", tt.body)
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

func TestDatasourceAPI_UpdateDatasource(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	r := gin.New()
	api := NewDatasourceAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
	})
	r.PUT("/datasource", api.UpdateDatasource)
	body := encoding.JSONMarshal(&model.Datasource{})

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
			name: "update data source failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				datasourceSrv.EXPECT().UpdateDatasource(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update data source successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				datasourceSrv.EXPECT().UpdateDatasource(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/datasource", tt.body)
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

func TestDatasourceAPI_DeleteDatasourceByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	r := gin.New()
	api := NewDatasourceAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
	})
	r.DELETE("/datasource/:uid", api.DeleteDatasourceByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete data source failure",
			prepare: func() {
				datasourceSrv.EXPECT().DeleteDatasourceByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete data source successfully",
			prepare: func() {
				datasourceSrv.EXPECT().DeleteDatasourceByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/datasource/1234", http.NoBody)
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

func TestDatasourceAPI_GetDatasourceByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	r := gin.New()
	api := NewDatasourceAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
	})
	r.GET("/datasource/:uid", api.GetDatasourceByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get data source failure",
			prepare: func() {
				datasourceSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get data source successfully",
			prepare: func() {
				datasourceSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "1234").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/datasource/1234", http.NoBody)
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

func TestDatasourceAPI_GetDatasources(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	datasourceSrv := service.NewMockDatasourceService(ctrl)
	r := gin.New()
	api := NewDatasourceAPI(&deps.API{
		DatasourceSrv: datasourceSrv,
	})
	r.GET("/datasource", api.GetDatasources)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get data sources failure",
			prepare: func() {
				datasourceSrv.EXPECT().GetDatasources(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get data sources successfully",
			prepare: func() {
				datasourceSrv.EXPECT().GetDatasources(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/datasource", http.NoBody)
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
