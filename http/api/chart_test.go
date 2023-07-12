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

	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func TestChartAPI_CreateChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	integrationSrv := service.NewMockIntegrationService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv:       chartSrv,
		IntegrationSrv: integrationSrv,
	})
	r.POST("/charts", api.CreateChart)
	body := encoding.JSONMarshal(&model.Chart{})

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
			name: "create chart failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().CreateChart(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "integration failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().CreateChart(gomock.Any(), gomock.Any()).Return("1234", nil)
				integrationSrv.EXPECT().DisconnectSource(gomock.Any(), gomock.Any(), model.ChartConnection).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create chart successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().CreateChart(gomock.Any(), gomock.Any()).Return("1234", nil)
				integrationSrv.EXPECT().DisconnectSource(gomock.Any(), gomock.Any(), model.ChartConnection).Return(nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/charts", tt.body)
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

func TestChartAPI_SearchCharts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv: chartSrv,
	})
	r.PUT("/charts", api.SearchCharts)
	params := encoding.JSONMarshal(&model.SearchChartRequest{})

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
			name: "search chart failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				chartSrv.EXPECT().SearchCharts(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search chart successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				chartSrv.EXPECT().SearchCharts(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/charts", tt.body)
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

func TestChartAPI_UpdateChart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	integrationSrv := service.NewMockIntegrationService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv:       chartSrv,
		IntegrationSrv: integrationSrv,
	})
	r.PUT("/chart", api.UpdateChart)
	body := encoding.JSONMarshal(&model.Chart{Integration: "abc"})

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
			name: "update chart failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().UpdateChart(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "integration failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().UpdateChart(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().ConnectSource(gomock.Any(), "abc", gomock.Any(), model.ChartConnection).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update chart successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				chartSrv.EXPECT().UpdateChart(gomock.Any(), gomock.Any()).Return(nil)
				integrationSrv.EXPECT().ConnectSource(gomock.Any(), "abc", gomock.Any(), model.ChartConnection).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/chart", tt.body)
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

func TestChartAPI_DeleteChartByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv: chartSrv,
	})
	r.DELETE("/chart/:uid", api.DeleteChartByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete chart failure",
			prepare: func() {
				chartSrv.EXPECT().DeleteChartByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete chart successfully",
			prepare: func() {
				chartSrv.EXPECT().DeleteChartByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/chart/1234", http.NoBody)
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

func TestChartAPI_GetChartByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv: chartSrv,
	})
	r.GET("/chart/:uid", api.GetChartByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get chart failure",
			prepare: func() {
				chartSrv.EXPECT().GetChartByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get chart successfully",
			prepare: func() {
				chartSrv.EXPECT().GetChartByUID(gomock.Any(), "1234").Return(&model.Chart{}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/chart/1234", http.NoBody)
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

func TestChartAPI_GetDashboardsByChartsUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		DashboardSrv: dashboardSrv,
	})
	r.GET("/chart/:uid/dashboards", api.GetDashboardsByChartUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get dashboards failure",
			prepare: func() {
				dashboardSrv.EXPECT().GetDashboardsByChartUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get dashboards successfully",
			prepare: func() {
				dashboardSrv.EXPECT().GetDashboardsByChartUID(gomock.Any(), "1234").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/chart/1234/dashboards", http.NoBody)
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

func TestChartAPI_UnlinkChartFromDashboard(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	chartSrv := service.NewMockChartService(ctrl)
	r := gin.New()
	api := NewChartAPI(&deps.API{
		ChartSrv: chartSrv,
	})
	r.DELETE("/charts/:uid/dashboards/:dashboardUID", api.UnlinkChartFromDashboard)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "unlink failure",
			prepare: func() {
				chartSrv.EXPECT().UnlinkChartFromDashboard(gomock.Any(), "1234", "abc").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "unlink successfully",
			prepare: func() {
				chartSrv.EXPECT().UnlinkChartFromDashboard(gomock.Any(), "1234", "abc").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/charts/1234/dashboards/abc", http.NoBody)
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
