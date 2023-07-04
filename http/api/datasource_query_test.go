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
	"github.com/lindb/linsight/plugin"
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

func TestDatasourceQueryAPI_DataQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsSrv := service.NewMockDatasourceService(ctrl)
	dsMrg := datasource.NewMockManager(ctrl)
	query := plugin.NewMockDatasourcePlugin(ctrl)
	r := gin.New()
	api := NewDatasourceQueryAPI(&deps.API{
		DatasourceSrv: dsSrv,
		DatasourceMgr: dsMrg,
	})
	r.PUT("/datasource/query", api.DataQuery)
	body := encoding.JSONMarshal(&model.QueryRequest{Queries: []*model.Query{{Datasource: model.TargetDatasource{UID: "uid"}}}})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "request bind failure",
			body: bytes.NewBuffer([]byte("bbc")),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get datasource failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get plugin failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "data query failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(query, nil)
				query.EXPECT().DataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "data query successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(query, nil)
				query.EXPECT().DataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/datasource/query", tt.body)
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

func TestDatasourceQueryAPI_MetadataQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dsSrv := service.NewMockDatasourceService(ctrl)
	dsMrg := datasource.NewMockManager(ctrl)
	query := plugin.NewMockDatasourcePlugin(ctrl)
	r := gin.New()
	api := NewDatasourceQueryAPI(&deps.API{
		DatasourceSrv: dsSrv,
		DatasourceMgr: dsMrg,
	})
	r.PUT("/datasource/query", api.MetadataQuery)
	body := encoding.JSONMarshal(&model.Query{Datasource: model.TargetDatasource{UID: "uid"}})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "request bind failure",
			body: bytes.NewBuffer([]byte("bbc")),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get datasource failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get plugin failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "metadata query failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(query, nil)
				query.EXPECT().MetadataQuery(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "metadata query successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dsSrv.EXPECT().GetDatasourceByUID(gomock.Any(), "uid").Return(&model.Datasource{}, nil)
				dsMrg.EXPECT().GetPlugin(gomock.Any()).Return(query, nil)
				query.EXPECT().MetadataQuery(gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/datasource/query", tt.body)
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
