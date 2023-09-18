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
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/alerting/types"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/util"
)

// BootAPI represents boot information related api handlers.
type BootAPI struct {
	deps *depspkg.API

	logger logger.Logger
}

// NewBootAPI creates a BootAPI instance.
func NewBootAPI(deps *depspkg.API) *BootAPI {
	return &BootAPI{
		deps:   deps,
		logger: logger.GetLogger("API", "Boot"),
	}
}

// Boot gets boot information after signed in.
func (api *BootAPI) Boot(c *gin.Context) {
	ctx := c.Request.Context()
	user := util.GetUser(ctx)
	// need read data from backend
	signedUser, err := api.deps.UserSrv.GetSignedUser(ctx, user.User.ID)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	boot := &model.BootData{
		Home:          signedUser.Preference.HomePage,
		User:          *signedUser,
		Notifications: types.GetReceiverPlugins(),
	}
	if signedUser.Org != nil {
		// if user belong a org, get datasource/nav for org level.
		datasources, err0 := api.deps.DatasourceSrv.GetDatasources(ctx)
		if err0 != nil {
			api.logger.Error("get datasources for current org fail", logger.Error(err0))
		} else {
			boot.Datasources = datasources
		}
		nav, err0 := api.deps.CmpSrv.GetComponentTreeByCurrentOrg(ctx)
		if err0 != nil {
			api.logger.Error("get nav tree for current org fail", logger.Error(err0))
		} else {
			boot.NavTree = nav
		}
	}

	integrations, err := api.deps.IntegrationSrv.GetIntegrations(ctx)
	if err != nil {
		api.logger.Error("get integrations fail", logger.Error(err))
	} else {
		boot.Integrations = integrations
	}

	httppkg.OK(c, boot)
}
