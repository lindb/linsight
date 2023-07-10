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
	"github.com/gin-gonic/gin"
	httppkg "github.com/lindb/common/pkg/http"

	depspkg "github.com/lindb/linsight/http/deps"
)

// IntegerationAPI represents integration information related api handlers.
type IntegerationAPI struct {
	deps *depspkg.API
}

// NewIntegrationAPI creates an IntegerationAPI instance.
func NewIntegrationAPI(deps *depspkg.API) *IntegerationAPI {
	return &IntegerationAPI{
		deps: deps,
	}
}

// GetIntegrations returns all supported integrations.
func (api *IntegerationAPI) GetIntegrations(c *gin.Context) {
	integrations, err := api.deps.IntegrationSrv.GetIntegrations(c.Request.Context())
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, integrations)
}
