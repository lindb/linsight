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
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
)

// NavAPI represents navigation related api handlers.
type NavAPI struct {
	deps *depspkg.API
}

// NewNavAPI creates a NavAPI instance.
func NewNavAPI(deps *depspkg.API) *NavAPI {
	return &NavAPI{
		deps: deps,
	}
}

// UpdateNav updates the navigation.
func (api *NavAPI) UpdateNav(c *gin.Context) {
	var nav model.Nav
	if err := c.ShouldBind(&nav); err != nil {
		httppkg.Error(c, err)
		return
	}
	ctx := c.Request.Context()
	signedUser := util.GetUser(ctx)
	err := api.deps.NavSrv.UpdateNav(ctx, signedUser.Org.ID, &nav)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, "Navigation updated")
}

// GetNav returns the navigation for current org.
func (api *NavAPI) GetNav(c *gin.Context) {
	ctx := c.Request.Context()
	signedUser := util.GetUser(ctx)
	nav, err := api.deps.NavSrv.GetNavByOrgID(ctx, signedUser.Org.ID)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, nav)
}
