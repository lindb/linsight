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
	"gorm.io/datatypes"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestDashboardAPI_CreateDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	chartSrv := service.NewMockChartService(ctrl)
	integrationSrv := service.NewMockIntegrationService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv:   dashboardSrv,
		ChartSrv:       chartSrv,
		IntegrationSrv: integrationSrv,
	})
	r.POST("/dashboard", api.CreateDashboard)
	body := encoding.JSONMarshal(&model.Dashboard{Title: "test"})

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
			name: "create dashboard failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dashboardSrv.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "link charts to dashboard failure",
			body: bytes.NewBuffer(encoding.JSONMarshal(&model.Dashboard{Config: body})),
			prepare: func() {
				dashboardSrv.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).Return("1234", nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "integration failure",
			body: bytes.NewBuffer(encoding.JSONMarshal(&model.Dashboard{Config: body})),
			prepare: func() {
				dashboardSrv.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).Return("1234", nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().DisconnectSource(gomock.Any(), gomock.Any(), model.DashboardResource).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create dashboard successfully",
			body: bytes.NewBuffer(encoding.JSONMarshal(&model.Dashboard{Config: body})),
			prepare: func() {
				dashboardSrv.EXPECT().CreateDashboard(gomock.Any(), gomock.Any()).Return("1234", nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().DisconnectSource(gomock.Any(), gomock.Any(), model.DashboardResource).Return(nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/dashboard", tt.body)
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

func TestDashboardAPI_UpdateDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	chartSrv := service.NewMockChartService(ctrl)
	integrationSrv := service.NewMockIntegrationService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv:   dashboardSrv,
		ChartSrv:       chartSrv,
		IntegrationSrv: integrationSrv,
	})
	r.PUT("/dashboard", api.UpdateDashboard)
	body := encoding.JSONMarshal(&model.Dashboard{
		Integration: "abc",
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
			name: "update dashboard failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dashboardSrv.EXPECT().UpdateDashboard(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "link chart to dashboard failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dashboardSrv.EXPECT().UpdateDashboard(gomock.Any(), gomock.Any()).Return(nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "integration failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dashboardSrv.EXPECT().UpdateDashboard(gomock.Any(), gomock.Any()).Return(nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().ConnectSource(gomock.Any(), "abc", gomock.Any(), model.DashboardResource).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update dashboard successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				dashboardSrv.EXPECT().UpdateDashboard(gomock.Any(), gomock.Any()).Return(nil)
				chartSrv.EXPECT().LinkChartsToDashboard(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().ConnectSource(gomock.Any(), "abc", gomock.Any(), model.DashboardResource).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/dashboard", tt.body)
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

func TestDashboardAPI_DeleteDashboardByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.DELETE("/dashboard/:uid", api.DeleteDashboardByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete dashboard failure",
			prepare: func() {
				dashboardSrv.EXPECT().DeleteDashboardByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete dashboard successfully",
			prepare: func() {
				dashboardSrv.EXPECT().DeleteDashboardByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/dashboard/1234", http.NoBody)
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

func TestDashboardAPI_GetDashboardByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.GET("/dashboard/:uid", api.GetDashboardByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get dashboard failure",
			prepare: func() {
				dashboardSrv.EXPECT().GetDashboardByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "unmarshal dashboard failure",
			prepare: func() {
				var dashboard datatypes.JSON
				_ = encoding.JSONUnmarshal([]byte("{}"), &dashboard)
				dashboardSrv.EXPECT().GetDashboardByUID(gomock.Any(), "1234").Return(&model.Dashboard{
					Config: dashboard,
				}, nil)
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return fmt.Errorf("err")
				}
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get provisioning dashboard failure",
			prepare: func() {
				var dashboard datatypes.JSON
				_ = encoding.JSONUnmarshal([]byte("{}"), &dashboard)
				dashboardSrv.EXPECT().GetDashboardByUID(gomock.Any(), "1234").Return(&model.Dashboard{
					Config: dashboard,
				}, nil)
				dashboardSrv.EXPECT().GetProvisioningDashboard(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get dashboard successfully",
			prepare: func() {
				var dashboard datatypes.JSON
				_ = encoding.JSONUnmarshal([]byte("{}"), &dashboard)
				dashboardSrv.EXPECT().GetDashboardByUID(gomock.Any(), "1234").Return(&model.Dashboard{
					Config: dashboard,
				}, nil)
				dashboardSrv.EXPECT().GetProvisioningDashboard(gomock.Any(), "1234").Return(&model.DashboardProvisioning{AllowUIUpdates: false}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
			}()
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/dashboard/1234", http.NoBody)
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

func TestDashboardAPI_SearchDashboards(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.PUT("/dashboard", api.SearchDashboards)
	params := encoding.JSONMarshal(&model.SearchDashboardRequest{})

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
			name: "search dashboard failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				dashboardSrv.EXPECT().SearchDashboards(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search dashboard successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				dashboardSrv.EXPECT().SearchDashboards(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/dashboard", tt.body)
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

func TestDashboardAPI_StarDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.PUT("/dashboard/:uid/star", api.StarDashboard)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "star dashboard failure",
			prepare: func() {
				dashboardSrv.EXPECT().StarDashboard(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "star dashboard successfully",
			prepare: func() {
				dashboardSrv.EXPECT().StarDashboard(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/dashboard/1234/star", http.NoBody)
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
func TestDashboardAPI_UnstarDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewDashboardAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.DELETE("/dashboard/:uid/star", api.UnstarDashboard)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "unstar dashboard failure",
			prepare: func() {
				dashboardSrv.EXPECT().UnstarDashboard(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "unstar dashboard successfully",
			prepare: func() {
				dashboardSrv.EXPECT().UnstarDashboard(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/dashboard/1234/star", http.NoBody)
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
