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

func buildNotificationTemplateAPI(ctrl *gomock.Controller) (
	*NotificationTemplateAPI,
	*service.MockNotificationTemplateService,
	*service.MockNotificationTemplateExampleService,
	*gin.Engine,
) {
	templateSrv := service.NewMockNotificationTemplateService(ctrl)
	exampleSrv := service.NewMockNotificationTemplateExampleService(ctrl)
	// No-op loader: return empty org templates so rendered output is template-only.
	renderingEngine := alerttemplate.NewRenderingEngine(func(int64) (string, error) { return "", nil }, 5*time.Minute)
	apiDeps := &deps.API{
		NotificationTemplateSrv:        templateSrv,
		NotificationTemplateExampleSrv: exampleSrv,
		RenderingEngine:                renderingEngine,
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
	return notifAPI, templateSrv, exampleSrv, r
}

// ─── NotificationTemplate CRUD ────────────────────────────────────────────────

func TestNotificationTemplateAPI_CreateNotificationTemplate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, templateSrv, _, r := buildNotificationTemplateAPI(ctrl)
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

	notifAPI, templateSrv, _, r := buildNotificationTemplateAPI(ctrl)
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

	notifAPI, templateSrv, _, r := buildNotificationTemplateAPI(ctrl)
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

	notifAPI, templateSrv, _, r := buildNotificationTemplateAPI(ctrl)
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

	notifAPI, templateSrv, _, r := buildNotificationTemplateAPI(ctrl)
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

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.POST("/templates/preview", notifAPI.PreviewNotificationTemplate)

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
			// No payload → DB has no default → fallback to hardcoded sample data containing "HighCPUUsage".
			name: "fallback to hardcoded sample data",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .CommonLabels.alertname }}`,
			})),
			prepare: func() {
				exampleSrv.EXPECT().GetDefaultNotificationTemplateExample(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "HighCPUUsage")
			},
		},
		{
			// No payload → DB default example is used for rendering.
			name: "use DB default example when no payload",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .CommonLabels.alertname }}`,
			})),
			prepare: func() {
				exampleSrv.EXPECT().GetDefaultNotificationTemplateExample(gomock.Any()).Return(
					&model.NotificationTemplateExample{
						Payload: `{"commonLabels":{"alertname":"DBDefaultAlert"}}`,
					}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "DBDefaultAlert")
			},
		},
		{
			// No payload → DB query fails → propagate error.
			name: "DB default fetch error",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .CommonLabels.alertname }}`,
			})),
			prepare: func() {
				exampleSrv.EXPECT().GetDefaultNotificationTemplateExample(gomock.Any()).Return(nil, fmt.Errorf("db error"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
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
			// Syntactically broken template (no payload) → GetDefault called, then render error.
			name: "invalid template syntax",
			body: bytes.NewBuffer(encoding.JSONMarshal(map[string]any{
				"template": `{{ .Foo`,
			})),
			prepare: func() {
				exampleSrv.EXPECT().GetDefaultNotificationTemplateExample(gomock.Any()).Return(nil, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/templates/preview", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

// ─── NotificationTemplateExample CRUD ─────────────────────────────────────────

func TestNotificationTemplateAPI_CreateNotificationTemplateExample(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.POST("/template-examples", notifAPI.CreateNotificationTemplateExample)

	body := encoding.JSONMarshal(&model.NotificationTemplateExample{Name: "ex-1", Payload: `{}`})

	cases := []struct {
		name    string
		body    io.Reader
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "create failure",
			body: bytes.NewBuffer(body),
			prepare: func() {
				exampleSrv.EXPECT().CreateNotificationTemplateExample(gomock.Any(), gomock.Any()).Return("", fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "create successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				exampleSrv.EXPECT().CreateNotificationTemplateExample(gomock.Any(), gomock.Any()).Return("ex-uid", nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Equal(t, `"ex-uid"`, resp.Body.String())
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPost, "/template-examples", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_UpdateNotificationTemplateExample(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.PUT("/template-examples", notifAPI.UpdateNotificationTemplateExample)

	body := encoding.JSONMarshal(&model.NotificationTemplateExample{UID: "ex-1", Name: "updated"})

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
				exampleSrv.EXPECT().UpdateNotificationTemplateExample(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "update successfully",
			body: bytes.NewBuffer(body),
			prepare: func() {
				exampleSrv.EXPECT().UpdateNotificationTemplateExample(gomock.Any(), gomock.Any()).Return(nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodPut, "/template-examples", tt.body)
			req.Header.Set("content-type", "application/json")
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_DeleteNotificationTemplateExampleByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.DELETE("/template-examples/:uid", notifAPI.DeleteNotificationTemplateExampleByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "delete failure",
			prepare: func() {
				exampleSrv.EXPECT().DeleteNotificationTemplateExampleByUID(gomock.Any(), "ex-1").Return(fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "delete successfully",
			prepare: func() {
				exampleSrv.EXPECT().DeleteNotificationTemplateExampleByUID(gomock.Any(), "ex-1").Return(nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodDelete, "/template-examples/ex-1", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_GetNotificationTemplateExampleByUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.GET("/template-examples/:uid", notifAPI.GetNotificationTemplateExampleByUID)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "get failure",
			prepare: func() {
				exampleSrv.EXPECT().GetNotificationTemplateExampleByUID(gomock.Any(), "ex-1").Return(nil, fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "get successfully",
			prepare: func() {
				exampleSrv.EXPECT().GetNotificationTemplateExampleByUID(gomock.Any(), "ex-1").
					Return(&model.NotificationTemplateExample{Name: "ex-1"}, nil)
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, resp.Code)
				assert.Contains(t, resp.Body.String(), "ex-1")
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/template-examples/ex-1", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}

func TestNotificationTemplateAPI_SearchNotificationTemplateExamples(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notifAPI, _, exampleSrv, r := buildNotificationTemplateAPI(ctrl)
	r.GET("/template-examples", notifAPI.SearchNotificationTemplateExamples)

	cases := []struct {
		name    string
		prepare func()
		assert  func(resp *httptest.ResponseRecorder)
	}{
		{
			name: "search failure",
			prepare: func() {
				exampleSrv.EXPECT().SearchNotificationTemplateExamples(gomock.Any(), gomock.Any()).Return(nil, int64(0), fmt.Errorf("err"))
			},
			assert: func(resp *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusInternalServerError, resp.Code)
			},
		},
		{
			name: "search successfully",
			prepare: func() {
				exampleSrv.EXPECT().SearchNotificationTemplateExamples(gomock.Any(), gomock.Any()).
					Return([]model.NotificationTemplateExample{{Name: "ex-1"}}, int64(1), nil)
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
			req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "/template-examples", http.NoBody)
			resp := httptest.NewRecorder()
			r.ServeHTTP(resp, req)
			tt.assert(resp)
		})
	}
}
