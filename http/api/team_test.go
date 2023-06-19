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
	httppkg "github.com/lindb/linsight/pkg/http"
	"github.com/lindb/linsight/service"
)

type MockQueryJSONBinder struct {
}

func (MockQueryJSONBinder) Name() string {
	return "mockQueryJSONBinder"
}

// Bind binds request data to obj.
func (MockQueryJSONBinder) Bind(req *http.Request, obj any) error {
	return fmt.Errorf("err")
}

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

func TestTeamAPI_GetTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.PUT("/teams/:uid", api.GetTeamMembers)
	params := encoding.JSONMarshal(&model.SearchTeamMemberRequest{})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "cannot get params",
			body: bytes.NewBuffer([]byte("{abc")),
			prepare: func() {
				httppkg.QueryJSONBind = &MockQueryJSONBinder{}
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search team members failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().GetTeamMembers(gomock.Any(), "1234", gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search team members successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().GetTeamMembers(gomock.Any(), "1234", gomock.Any()).Return(nil, int64(0), nil)
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
				httppkg.QueryJSONBind = httppkg.QueryJSONBinder{}
			}()
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/teams/1234", tt.body)
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

func TestTeamAPI_AddTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.POST("/team/:uid", api.AddTeamMembers)
	params := encoding.JSONMarshal(&model.AddTeamMember{UserUIDs: []string{"1"}, Permission: model.PermissionAdmin})

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
			name: "add team member failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().AddTeamMembers(gomock.Any(), "1234", gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "add team member successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().AddTeamMembers(gomock.Any(), "1234", gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/team/1234", tt.body)
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

func TestTeamAPI_UpdateTeamMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.PUT("/team/:uid", api.UpdateTeamMember)
	params := encoding.JSONMarshal(&model.UpdateTeamMember{UserUID: "1", Permission: model.PermissionAdmin})

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
			name: "update team member failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().UpdateTeamMember(gomock.Any(), "1234", gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update team member successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().UpdateTeamMember(gomock.Any(), "1234", gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/team/1234", tt.body)
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

func TestTeamAPI_RemoveTeamMembers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	teamSrv := service.NewMockTeamService(ctrl)
	r := gin.New()
	api := NewTeamAPI(&deps.API{
		TeamSrv: teamSrv,
	})
	r.PUT("/team/:uid", api.RemoveTeamMember)
	params := encoding.JSONMarshal(&model.RemoveTeamMember{UserUIDs: []string{"1"}})

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
			name: "remove team member failure",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().RemoveTeamMember(gomock.Any(), "1234", gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "remove team member successfully",
			body: bytes.NewBuffer(params),
			prepare: func() {
				teamSrv.EXPECT().RemoveTeamMember(gomock.Any(), "1234", gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/team/1234", tt.body)
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
