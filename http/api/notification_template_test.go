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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/common/pkg/encoding"

	alerttemplate "github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// buildNotificationTemplateAPI constructs a test API instance.
// examples is loaded directly into deps.NotificationTemplateExamples (static slice, no service mock).
func buildNotificationTemplateAPI(ctrl *gomock.Controller, examples ...model.NotificationTemplateExample) (
	*NotificationTemplateAPI,
	*service.MockNotificationTemplateService,
	*gin.Engine,
) {
	templateSrv := service.NewMockNotificationTemplateService(ctrl)
	// No-op loader: return empty org templates so rendered output is template-only.
	renderingEngine := alerttemplate.NewRenderingEngine(func(int64) (string, error) { return "", nil }, 5*time.Minute)
	apiDeps := &deps.API{
		NotificationTemplateSrv:      templateSrv,
		NotificationTemplateExamples: examples,
		RenderingEngine:              renderingEngine,
	}
	notifAPI := NewNotificationTemplateAPI(apiDeps)
	r := gin.New()
	// Inject a dummy signed user so handlers that call util.GetUser never encounter a nil user.
	// orgID=0 is sufficient for tests: Invalidate(0) is a no-op when nothing is cached.
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), constant.LinSightSignedKey, &model.SignedUser{
			Org: &model.Org{},
		})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	return notifAPI, templateSrv, r
}

// ─── NotificationTemplate CRUD ────────────────────────────────────────────────

func TestNotificationTemplateAPI_CreateNotificationTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, r := buildNotificationTemplateAPI(ctrl)
	r.POST("/templates", notifAPI.CreateNotificationTemplate)

	body := encoding.JSONMarshal(&model.NotificationTemplate{Name: "tmpl-1", Template: "hello"})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "cannot bind params",
			body: http.NoBody,
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				templateSrv.EXPECT().CreateNotificationTemplate(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				templateSrv.EXPECT().CreateNotificationTemplate(gomock.Any(), gomock.Any()).Return("uid-1", nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Equal(t, `"uid-1"`, resp.Body.String())
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/templates", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_UpdateNotificationTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, r := buildNotificationTemplateAPI(ctrl)
	r.PUT("/templates", notifAPI.UpdateNotificationTemplate)

	body := encoding.JSONMarshal(&model.NotificationTemplate{UID: "uid-1", Name: "updated"})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "update failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				templateSrv.EXPECT().UpdateNotificationTemplate(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				templateSrv.EXPECT().UpdateNotificationTemplate(gomock.Any(), gomock.Any()).Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/templates", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_DeleteNotificationTemplateByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, r := buildNotificationTemplateAPI(ctrl)
	r.DELETE("/templates/:uid", notifAPI.DeleteNotificationTemplateByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete failure",
			prepare: func() {
				templateSrv.EXPECT().DeleteNotificationTemplateByUID(gomock.Any(), "uid-1").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete successfully",
			prepare: func() {
				templateSrv.EXPECT().DeleteNotificationTemplateByUID(gomock.Any(), "uid-1").Return(nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/templates/uid-1", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_GetNotificationTemplateByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, r := buildNotificationTemplateAPI(ctrl)
	r.GET("/templates/:uid", notifAPI.GetNotificationTemplateByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get failure",
			prepare: func() {
				templateSrv.EXPECT().GetNotificationTemplateByUID(gomock.Any(), "uid-1").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get successfully",
			prepare: func() {
				templateSrv.EXPECT().GetNotificationTemplateByUID(gomock.Any(), "uid-1").
					Return(&model.NotificationTemplate{Name: "tmpl"}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "tmpl")
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/templates/uid-1", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_SearchNotificationTemplates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, r := buildNotificationTemplateAPI(ctrl)
	r.GET("/templates", notifAPI.SearchNotificationTemplates)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "search failure",
			prepare: func() {
				templateSrv.EXPECT().SearchNotificationTemplates(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search successfully",
			prepare: func() {
				templateSrv.EXPECT().SearchNotificationTemplates(gomock.Any(), gomock.Any()).
					Return([]model.NotificationTemplate{{Name: "t1"}}, int64(1), nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), `"total":1`)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/templates", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

// ─── Preview ──────────────────────────────────────────────────────────────────

func TestNotificationTemplateAPI_PreviewNotificationTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// defaultExample is the built-in example used for tests that rely on slice-based default lookup.
	defaultExample := model.NotificationTemplateExample{
		IsDefault: true,
		Payload:   `{"commonLabels":{"alertname":"DBDefaultAlert"}}`,
	}

	cases := []struct {
		name     string
		examples []model.NotificationTemplateExample // contents of deps.NotificationTemplateExamples
		body     io.Reader
		assert   func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "cannot bind params",
			body: http.NoBody,
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			// No payload + no default in slice → fallback to hardcoded sample containing "HighCPUUsage".
			name: "fallback to hardcoded sample data",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .CommonLabels.alertname }}`,
			})),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "HighCPUUsage")
			},
		},
		{
			// No payload → slice contains a default example → it is used for rendering.
			name:     "use default example from slice when no payload",
			examples: []model.NotificationTemplateExample{defaultExample},
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .CommonLabels.alertname }}`,
			})),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "DBDefaultAlert")
			},
		},
		{
			// Payload is a JSON object → alerts auto-derived and rendered correctly.
			name: "inline payload renders correctly",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ range .FiringAlerts }}{{ .RuleTitle }}{{ end }}`,
				"payload":  json.RawMessage(`{"status":"firing","alerts":[{"ruleTitle":"DiskFull","state":2}]}`),
			})),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "DiskFull")
			},
		},
		{
			// Malformed JSON in the request body → ShouldBind fails with 500.
			name: "inline payload invalid json",
			body: bytes.NewBufferString(`{"template":"{{ .Status }}","payload":{not-valid-json}}`),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			// Syntactically broken template + no default in slice → render error.
			name: "invalid template syntax",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .Foo`,
			})),
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			// Each case builds its own API with the specific example slice for that scenario.
			notifAPI, _, r := buildNotificationTemplateAPI(ctrl, tt.examples...)
			r.POST("/templates/preview", notifAPI.PreviewNotificationTemplate)

			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/templates/preview", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

// ─── NotificationTemplateExample read-only handlers ───────────────────────────

func TestNotificationTemplateAPI_SearchNotificationTemplateExamples(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	examples := []model.NotificationTemplateExample{
		{Name: "Alpha Example"},
		{Name: "Beta Example"},
	}

	notifAPI, _, r := buildNotificationTemplateAPI(ctrl, examples...)
	r.GET("/template-examples", notifAPI.SearchNotificationTemplateExamples)

	cases := []struct {
		name   string
		query  string
		assert func(resp *httptest.ResponseRecorder)
	}{
		{
			name:  "search all",
			query: "",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), `"total":2`)
				assert.Contains(t, resp.Body.String(), "Alpha Example")
				assert.Contains(t, resp.Body.String(), "Beta Example")
			},
		},
		{
			name:  "search by name filter",
			query: "?name=Alpha",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), `"total":1`)
				assert.Contains(t, resp.Body.String(), "Alpha Example")
			},
		},
		{
			name:  "search no match",
			query: "?name=ZZZ",
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), `"total":0`)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/template-examples"+tt.query, http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}
