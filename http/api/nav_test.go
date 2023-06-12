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

	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

var ctx = context.WithValue(context.TODO(), constant.LinSightSignedKey, &model.SignedUser{
	Org: &model.Org{
		BaseModel: model.BaseModel{
			ID: 12,
		},
	},
	User: &model.User{
		BaseModel: model.BaseModel{
			ID: 10,
		},
	},
})

func TestNavAPI_UpdateNav(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	navSrv := service.NewMockNavService(ctrl)
	r := gin.New()
	api := NewNavAPI(&deps.API{
		NavSrv: navSrv,
	})
	r.PUT("/org/nav", api.UpdateNav)
	body := encoding.JSONMarshal(&model.Org{})

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
			name: "update nav failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				navSrv.EXPECT().UpdateNav(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update nav successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				navSrv.EXPECT().UpdateNav(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodPut, "/org/nav", tt.body)
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

func TestNavAPI_GetNavByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	navSrv := service.NewMockNavService(ctrl)
	r := gin.New()
	api := NewNavAPI(&deps.API{
		NavSrv: navSrv,
	})
	r.GET("/org/nav", api.GetNav)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get nav failure",
			prepare: func() {
				navSrv.EXPECT().GetNavByOrgID(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get nav successfully",
			prepare: func() {
				navSrv.EXPECT().GetNavByOrgID(gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/org/nav", http.NoBody)
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
