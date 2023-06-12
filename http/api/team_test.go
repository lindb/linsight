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

func TestTeamAPI_CreateTeam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.POST("/teams", api.CreateTeam)
	body := encoding.JSONMarshal(&model.Team{})

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
			name: "create team failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				teamSrv.EXPECT().CreateTeam(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create team successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				teamSrv.EXPECT().CreateTeam(gomock.Any(), gomock.Any()).Return("1234", nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/teams", tt.body)
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

func TestTeamAPI_SearchTeams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.PUT("/teams", api.SearchTeams)
	params := encoding.JSONMarshal(&model.SearchTeamRequest{})

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
			name: "search team failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().SearchTeams(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search team successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().SearchTeams(gomock.Any(), gomock.Any()).Return(nil, int64(0), nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/teams", tt.body)
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

func TestTeamAPI_UpdateTeam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.PUT("/team", api.UpdateTeam)
	body := encoding.JSONMarshal(&model.Team{})

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
			name: "update team failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				teamSrv.EXPECT().UpdateTeam(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update team successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				teamSrv.EXPECT().UpdateTeam(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/team", tt.body)
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

func TestTeamAPI_DeleteTeamByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.DELETE("/team/:uid", api.DeleteTeamByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete team failure",
			prepare: func() {
				teamSrv.EXPECT().DeleteTeamByUID(gomock.Any(), "1234").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete team successfully",
			prepare: func() {
				teamSrv.EXPECT().DeleteTeamByUID(gomock.Any(), "1234").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/team/1234", http.NoBody)
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

func TestTeamAPI_GetTeamByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.GET("/team/:uid", api.GetTeamByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get team failure",
			prepare: func() {
				teamSrv.EXPECT().GetTeamByUID(gomock.Any(), "1234").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get team successfully",
			prepare: func() {
				teamSrv.EXPECT().GetTeamByUID(gomock.Any(), "1234").Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/team/1234", http.NoBody)
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
