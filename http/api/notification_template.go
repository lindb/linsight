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
	"encoding/json"
	"strings"
	texttemplate "text/template"

	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/constant"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
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
	// Invalidate cached org templates so the next render picks up the new template.
	api.deps.RenderingEngine.Invalidate(util.GetUser(c.Request.Context()).Org.ID)
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
	// Invalidate cached org templates so the next render picks up the updated template.
	api.deps.RenderingEngine.Invalidate(util.GetUser(c.Request.Context()).Org.ID)
	httppkg.OK(c, "Notification template updated")
}

// DeleteNotificationTemplateByUID removes a template.
func (api *NotificationTemplateAPI) DeleteNotificationTemplateByUID(c *gin.Context) {
	uid := c.Param(constant.UID)
	if err := api.deps.NotificationTemplateSrv.DeleteNotificationTemplateByUID(c.Request.Context(), uid); err != nil {
		httppkg.Error(c, err)
		return
	}
	// Invalidate cached org templates so the deletion is reflected immediately on the next render.
	api.deps.RenderingEngine.Invalidate(util.GetUser(c.Request.Context()).Org.ID)
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

// PreviewNotificationTemplate renders a template string with org-level named blocks and optional
// payload data, using the same pipeline as alert dispatch (org templates + req.Template merged).
// If payload is supplied it is deserialized into NotificationData; otherwise hardcoded sample
// data is used so callers can try templates without constructing a full payload.
func (api *NotificationTemplateAPI) PreviewNotificationTemplate(c *gin.Context) {
	req := &struct {
		Template string          `json:"template"`
		Payload  json.RawMessage `json:"payload"` // optional: JSON object of NotificationData
	}{}
	if err := c.ShouldBind(req); err != nil {
		httppkg.Error(c, err)
		return
	}

	var data *template.NotificationData
	if len(req.Payload) > 0 {
		var err error
		data, err = template.BuildNotificationDataFromPayload(string(req.Payload))
		if err != nil {
			httppkg.Error(c, err)
			return
		}
	} else {
		// Try the org's designated default example first so users get a meaningful preview
		// without having to manually construct a payload.
		ex, err := api.deps.NotificationTemplateExampleSrv.GetDefaultNotificationTemplateExample(c.Request.Context())
		if err != nil {
			httppkg.Error(c, err)
			return
		}
		if ex != nil && ex.Payload != "" {
			data, err = template.BuildNotificationDataFromPayload(ex.Payload)
			if err != nil {
				httppkg.Error(c, err)
				return
			}
		} else {
			// fallback: use hardcoded sample data so the preview always produces visible output
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
	}

	// Use the shared rendering engine so org {{define}} blocks are available to req.Template,
	// mirroring exactly the pipeline used during alert dispatch.
	orgID := util.GetUser(c.Request.Context()).Org.ID
	result, err := api.deps.RenderingEngine.Render(orgID, req.Template, data)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"result": result})
}

// TemplateDefinition represents a named {{define "NAME"}} block extracted from org notification templates.
type TemplateDefinition struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// GetTemplateDefinitions returns all named {{define}} blocks from all org-level notification templates.
// The result can be used by a frontend picker so that users select a named template block
// for per-channel fields (title / body / subject / content) instead of typing raw template strings.
func (api *NotificationTemplateAPI) GetTemplateDefinitions(c *gin.Context) {
	templates, err := api.deps.NotificationTemplateSrv.GetAllNotificationTemplates(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	// Merge all template bodies so every {{define}} block is parsed in one pass.
	var sb strings.Builder
	for _, t := range templates {
		sb.WriteString(t.Template)
		sb.WriteByte('\n')
	}
	merged := sb.String()
	if strings.TrimSpace(merged) == "" {
		httppkg.OK(c, []TemplateDefinition{})
		return
	}

	// Parse the merged content; named sub-templates become enumerable via tmpl.Templates().
	tmpl, err := texttemplate.New("root").Parse(merged)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	seen := make(map[string]struct{})
	var defs []TemplateDefinition
	for _, t := range tmpl.Templates() {
		name := t.Name()
		// Skip the anonymous root and any template without a parsed tree.
		if name == "root" || t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		defs = append(defs, TemplateDefinition{
			Name:    name,
			Content: strings.TrimSpace(t.Tree.Root.String()),
		})
	}
	if defs == nil {
		defs = []TemplateDefinition{}
	}
	httppkg.OK(c, defs)
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
