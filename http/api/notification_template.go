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
	"context"
	"time"

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
)

// NotificationTemplateAPI handles HTTP requests for notification template management.
type NotificationTemplateAPI struct {
	deps *depspkg.API
}

// NewNotificationTemplateAPI creates a NotificationTemplateAPI instance.
func NewNotificationTemplateAPI(deps *depspkg.API) *NotificationTemplateAPI {
	return &NotificationTemplateAPI{deps: deps}
}

// CreateNotificationTemplate creates a new template.
func (api *NotificationTemplateAPI) CreateNotificationTemplate(c *gin.Context) {
	tmpl := &model.NotificationTemplate{}
	if err := c.ShouldBind(tmpl); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.NotificationTemplateSrv.CreateNotificationTemplate(c.Request.Context(), tmpl)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateNotificationTemplate updates an existing template.
func (api *NotificationTemplateAPI) UpdateNotificationTemplate(c *gin.Context) {
	tmpl := &model.NotificationTemplate{}
	if err := c.ShouldBind(tmpl); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.NotificationTemplateSrv.UpdateNotificationTemplate(c.Request.Context(), tmpl); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification template updated")
}

// DeleteNotificationTemplateByUID removes a template.
func (api *NotificationTemplateAPI) DeleteNotificationTemplateByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.NotificationTemplateSrv.DeleteNotificationTemplateByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification template deleted")
}

// GetNotificationTemplateByUID returns a template by UID.
func (api *NotificationTemplateAPI) GetNotificationTemplateByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	tmpl, err := api.deps.NotificationTemplateSrv.GetNotificationTemplateByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, tmpl)
}

// SearchNotificationTemplates queries templates with optional filters.
func (api *NotificationTemplateAPI) SearchNotificationTemplates(c *gin.Context) {
	req := &model.SearchNotificationTemplateRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	templates, total, err := api.deps.NotificationTemplateSrv.SearchNotificationTemplates(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":     total,
		"templates": templates,
	})
}

// PreviewNotificationTemplate renders a template string with the provided payload data and returns the result.
// Priority for data source: inline Payload > ExampleUID (loaded from DB) > hardcoded sample data (fallback).
func (api *NotificationTemplateAPI) PreviewNotificationTemplate(c *gin.Context) {
	req := &struct {
		Template   string `json:"template"`
		Payload    string `json:"payload"`    // optional: inline JSON NotificationData
		ExampleUID string `json:"exampleUID"` // optional: UID of a stored NotificationTemplateExample
	}{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}

	var data *template.NotificationData
	var buildErr error

	switch {
	case req.Payload != "":
		// highest priority: inline payload supplied by the caller
		data, buildErr = template.BuildNotificationDataFromPayload(req.Payload)
		if buildErr != nil {
			httppkg.Error(c, buildErr)
			return
		}

	case req.ExampleUID != "":
		// load the stored example from DB and build data from its payload
		ex, err := api.deps.NotificationTemplateExampleSrv.GetNotificationTemplateExampleByUID(c.Request.Context(), req.ExampleUID)
		if err != nil {
			httppkg.Error(c, err)
			return
		}
		data, buildErr = template.BuildNotificationDataFromPayload(ex.Payload)
		if buildErr != nil {
			httppkg.Error(c, buildErr)
			return
		}

	default:
		// backward-compatible fallback: use hardcoded sample data
		data = &template.NotificationData{
			Status:       "firing",
			ReceiverName: "preview",
			CommonLabels: map[string]string{
				"alertname": "HighCPUUsage",
				"instance":  "web-server-01",
				"severity":  "critical",
			},
			CommonAnnotations: map[string]string{
				"summary": "CPU usage is above 90%",
			},
			GroupLabels: map[string]string{
				"alertname": "HighCPUUsage",
			},
			ExternalURL: "http://localhost:3000",
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	renderer := template.NewRenderer()
	result, err := template.RenderWithContext(ctx, renderer, req.Template, data)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"result": result})
}

// ---- NotificationTemplateExample handlers ----

// CreateNotificationTemplateExample creates a new template example.
func (api *NotificationTemplateAPI) CreateNotificationTemplateExample(c *gin.Context) {
	ex := &model.NotificationTemplateExample{}
	if err := c.ShouldBind(ex); err != nil {
		httppkg.Error(c, err)
		return
	}
	uid, err := api.deps.NotificationTemplateExampleSrv.CreateNotificationTemplateExample(c.Request.Context(), ex)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, uid)
}

// UpdateNotificationTemplateExample updates an existing template example.
func (api *NotificationTemplateAPI) UpdateNotificationTemplateExample(c *gin.Context) {
	ex := &model.NotificationTemplateExample{}
	if err := c.ShouldBind(ex); err != nil {
		httppkg.Error(c, err)
		return
	}
	if err := api.deps.NotificationTemplateExampleSrv.UpdateNotificationTemplateExample(c.Request.Context(), ex); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification template example updated")
}

// DeleteNotificationTemplateExampleByUID removes a template example.
func (api *NotificationTemplateAPI) DeleteNotificationTemplateExampleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.NotificationTemplateExampleSrv.DeleteNotificationTemplateExampleByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Notification template example deleted")
}

// GetNotificationTemplateExampleByUID returns a template example by UID.
func (api *NotificationTemplateAPI) GetNotificationTemplateExampleByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	ex, err := api.deps.NotificationTemplateExampleSrv.GetNotificationTemplateExampleByUID(c.Request.Context(), uid)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, ex)
}

// SearchNotificationTemplateExamples queries template examples with optional filters.
func (api *NotificationTemplateAPI) SearchNotificationTemplateExamples(c *gin.Context) {
	req := &model.SearchNotificationTemplateExampleRequest{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}
	examples, total, err := api.deps.NotificationTemplateExampleSrv.SearchNotificationTemplateExamples(c.Request.Context(), req)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{
		"total":    total,
		"examples": examples,
	})
}
