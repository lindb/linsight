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

package integrations

import (
	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"
)

// Handler handles HTTP requests for the integrations API.
type Handler struct {
	svc CatalogService
}

// NewHandler creates a new Handler.
func NewHandler(svc CatalogService) *Handler {
	return &Handler{svc: svc}
}

// GetCategories returns all categories and their integration counts.
func (h *Handler) GetCategories(c *gin.Context) {
	categories, err := h.svc.ListCategories(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, gin.H{"categories": categories})
}

// SearchIntegrations searches for integrations by category and query string.
func (h *Handler) SearchIntegrations(c *gin.Context) {
	category := c.Query("category")
	q := c.Query("q")

	integrations, err := h.svc.SearchIntegrations(c.Request.Context(), category, q)
	if err != nil {
		httppkg.Error(c, err)
		return
	}

	httppkg.OK(c, gin.H{"integrations": integrations, "total": len(integrations)})
}

// GetIntegration retrieves a specific integration by ID.
func (h *Handler) GetIntegration(c *gin.Context) {
	id := c.Param("id")

	integ, err := h.svc.GetIntegration(c.Request.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			c.JSON(404, gin.H{"error": "integration not found"})
			return
		}
		httppkg.Error(c, err)
		return
	}

	httppkg.OK(c, integ)
}
