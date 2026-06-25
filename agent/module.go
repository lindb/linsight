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

// Package agent is the AI Agent sub-module entry point.
// Import as: agentmodule "github.com/lindb/linsight/agent"
package agent

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/accesscontrol"
	agenthandler "github.com/lindb/linsight/agent/handler"
	modelcomp "github.com/lindb/linsight/agent/components/model"
	skillscomp "github.com/lindb/linsight/agent/components/skills"
	toolscomp "github.com/lindb/linsight/agent/components/tools"
	agentmodel "github.com/lindb/linsight/agent/model"
	agentservice "github.com/lindb/linsight/agent/service"
	"github.com/lindb/linsight/agent/telemetry"
	"github.com/lindb/linsight/config"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/http/middleware"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

// agentModule holds the handler wired up during Bootstrap.
// A nil mod means the agent is disabled or Bootstrap was never called.
type agentModule struct {
	handler *agenthandler.AgentHandler
}

// mod is the package-level singleton; nil if agent is disabled.
var mod *agentModule

// Bootstrap wires up all agent services, runs DB migrations, and registers the
// OTel observability handler. It must be called once from cmd/main.go's buildDeps()
// before the server starts accepting requests. If cfg.Agent is nil or Enabled is
// false this function is a no-op, so callers need no guard code.
func Bootstrap(cfg *config.Server, db dbpkg.DB) {
	if cfg.Agent == nil || !cfg.Agent.Enabled {
		return
	}

	// Idempotent schema migration — runs on every startup, safe to repeat.
	db.RawDB().AutoMigrate(
		&agentmodel.AgentModel{},
		&agentmodel.AgentOrgModelConfig{},
		&agentmodel.AgentMCPServer{},
		&agentmodel.AgentConversation{},
		&agentmodel.AgentMessage{},
		&agentmodel.AgentSkill{},
		&agentmodel.AgentOrgSkill{},
		&agentmodel.AgentConversationSkill{},
	)

	// Wire service dependencies bottom-up.
	settingsSrv := agentservice.NewAgentSettingsService(db)
	modelFactory := modelcomp.NewModelFactory(settingsSrv)
	proxy := modelcomp.NewDynamicModelProxy(modelFactory)
	toolsFactory := toolscomp.NewMCPToolsFactory(cfg.Agent, settingsSrv)
	skillRegistry := skillscomp.NewSkillRegistry(db, toolsFactory)
	agentSrv, err := agentservice.NewAgentService(cfg.Agent, settingsSrv, modelFactory, skillRegistry, proxy)
	if err != nil {
		panic(fmt.Sprintf("agent: failed to compile react graph: %v", err))
	}

	// Register the Eino global callback that annotates each component with org/user IDs.
	telemetry.Register()

	mod = &agentModule{
		handler: agenthandler.NewAgentHandler(cfg.Agent, agentSrv, settingsSrv),
	}
}

// RegisterRoutes attaches all agent HTTP routes to router using the project's
// standard middleware.Authorize pattern. It must be called from http/router.go's
// RegisterRouters(). If Bootstrap was not called or the agent is disabled (mod == nil)
// this function is a no-op.
func RegisterRoutes(router *gin.RouterGroup, deps *depspkg.API) {
	if mod == nil {
		return
	}
	h := mod.handler

	// Viewer-level routes: any authenticated user may invoke the agent.
	router.POST("/agent/chat",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, h.Chat)...)
	router.POST("/agent/conversations/:uid/chat",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, h.Chat)...)
	router.GET("/agent/conversations",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, h.ListConversations)...)
	router.DELETE("/agent/conversations/:uid",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, h.DeleteConversation)...)

	// Admin-level routes: only org admins may configure the agent.

	// Model CRUD — /agent/models/default must be registered before /agent/models/:uid
	// so that Gin's exact-match rule picks up the literal "default" segment first.
	router.GET("/agent/models",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, h.ListModels)...)
	router.POST("/agent/models",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.CreateModel)...)
	router.GET("/agent/models/default",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, h.GetDefaultModel)...)
	router.GET("/agent/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, h.GetModel)...)
	router.PUT("/agent/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.UpdateModel)...)
	router.DELETE("/agent/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.DeleteModel)...)
	router.PUT("/agent/models/:uid/default",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.SetDefaultModel)...)

	// Routing config
	router.GET("/agent/settings/model-routing",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, h.GetOrgModelConfig)...)
	router.PUT("/agent/settings/model-routing",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.SaveOrgModelConfig)...)
	router.GET("/agent/settings/mcp",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, h.ListMCPServers)...)
	router.POST("/agent/settings/mcp",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.CreateMCPServer)...)
	router.DELETE("/agent/settings/mcp/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, h.DeleteMCPServer)...)
}
