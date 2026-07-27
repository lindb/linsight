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
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/lindb/linsight/accesscontrol"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/http/middleware"
)

// module holds the integrations module singleton.
type module struct {
	handler *Handler
}

var (
	mod        *module
	modInit    sync.Once
	catalogSvc CatalogService = noopCatalogService{}
)

// Bootstrap initializes the integrations module.
func Bootstrap(cfg *Config) error {
	var initErr error
	modInit.Do(func() {
		var svc CatalogService

		if cfg != nil && cfg.Dir != "" {
			store, err := NewStore(cfg.Dir)
			if err != nil {
				initErr = err
				return
			}
			svc = NewCatalogService(store)
		} else {
			svc = noopCatalogService{}
		}

		catalogSvc = svc
		mod = &module{
			handler: NewHandler(svc),
		}
	})
	return initErr
}

// GetCatalogService returns the global CatalogService singleton.
// It may return noopCatalogService if Bootstrap was not called or no integrations directory was configured.
func GetCatalogService() CatalogService {
	return catalogSvc
}

// RegisterRoutes registers all integrations routes.
func RegisterRoutes(router gin.IRouter, deps *depspkg.API) {
	if mod == nil {
		return
	}

	// All routes are read-only and require Viewer access
	router.GET(
		"/integrations/categories",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, mod.handler.GetCategories)...,
	)
	router.GET(
		"/integrations",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, mod.handler.SearchIntegrations)...,
	)
	router.GET(
		"/integrations/:id",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, mod.handler.GetIntegration)...,
	)
}
