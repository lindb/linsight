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

// Package ai is the AI Agent sub-module entry point.
// Import as: aimodule "github.com/lindb/linsight/ai"
package ai

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/lindb/linsight/accesscontrol"
	agentcomp "github.com/lindb/linsight/ai/components/agent"
	memorycomp "github.com/lindb/linsight/ai/components/memory"
	modelcomp "github.com/lindb/linsight/ai/components/model"
	skillscomp "github.com/lindb/linsight/ai/components/skills"
	toolscomp "github.com/lindb/linsight/ai/components/tools"
	builtintools "github.com/lindb/linsight/ai/components/tools/builtin"
	agenthandler "github.com/lindb/linsight/ai/handler"
	agentmodel "github.com/lindb/linsight/ai/model"
	agentservice "github.com/lindb/linsight/ai/service"
	"github.com/lindb/linsight/ai/telemetry"
	"github.com/lindb/linsight/config"
	depspkg "github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/http/middleware"
	dbpkg "github.com/lindb/linsight/pkg/db"
	datasourcepkg "github.com/lindb/linsight/plugin/datasource"
	servicepkg "github.com/lindb/linsight/service"
)

// ObservabilityDeps carries the observability service dependencies required by
// the built-in AI tools. Passed to Bootstrap so the AI module can access
// datasources and dashboards without creating duplicate service instances.
type ObservabilityDeps struct {
	DatasourceSrv servicepkg.DatasourceService
	DashboardSrv  servicepkg.DashboardService
	DatasourceMgr datasourcepkg.Manager
	// AlertRuleSrv backs the linsight_create_alert_rule tool (approval-gated).
	AlertRuleSrv servicepkg.AlertRuleService
}

// aiModule holds the domain handlers wired up during Bootstrap.
// A nil mod means the agent is disabled or Bootstrap was never called.
type aiModule struct {
	chat     *agenthandler.ChatHandler
	model    *agenthandler.ModelHandler
	mcp      *agenthandler.MCPHandler
	feedback *agenthandler.FeedbackHandler
}

// mod is the package-level singleton; nil if AI is disabled.
var mod *aiModule

// builtinSkillNames are the tool names seeded into the skills table on every
// startup with is_global=true so they pass the three-level governance funnel
// without any org subscription or per-conversation enablement.
var builtinSkillNames = []string{
	"linsight_list_datasources",
	"linsight_datasource_query",
	"linsight_search_dashboards",
	"linsight_create_alert_rule",
}

// Bootstrap wires up all agent services, runs DB migrations, and registers the
// OTel observability handler. It must be called once from cmd/main.go's buildDeps()
// before the server starts accepting requests. If cfg.AI is nil or Enabled is
// false this function is a no-op, so callers need no guard code.
func Bootstrap(cfg *config.Server, db dbpkg.DB, obsDeps ObservabilityDeps) {
	if cfg.AI == nil || !cfg.AI.Enabled {
		return
	}

	// Idempotent schema migration — runs on every startup, safe to repeat.
	// Fail fast: a migration error here means the agent cannot operate correctly.
	if err := db.RawDB().AutoMigrate(
		&agentmodel.Model{},
		&agentmodel.OrgModelConfig{},
		&agentmodel.MCP{},
		&agentmodel.Conversation{},
		&agentmodel.Message{},
		&agentmodel.Skill{},
		&agentmodel.OrgSkill{},
		&agentmodel.ConversationSkill{},
		&agentmodel.MessageFeedback{},
		&agentmodel.Event{},
		&agentmodel.InterruptRecord{},
	); err != nil {
		panic(fmt.Sprintf("ai: failed to migrate database schema: %v", err))
	}

	// Seed built-in skill entries so they pass the three-level governance funnel
	// without requiring org subscription or per-conversation enablement.
	// FirstOrCreate is idempotent: a second startup is a no-op.
	for _, name := range builtinSkillNames {
		db.RawDB().Where(agentmodel.Skill{Name: name}).
			Attrs(agentmodel.Skill{IsGlobal: true, SkillType: "builtin"}).
			FirstOrCreate(&agentmodel.Skill{})
	}

	// Wire service dependencies bottom-up.
	modelSrv := agentservice.NewModelService(db)
	mcpSrv := agentservice.NewMCPService(db)
	convSrv := agentservice.NewConversationService(db)
	feedbackSrv := agentservice.NewFeedbackService(db, convSrv)
	modelFactory := modelcomp.NewModelFactory(modelSrv)
	proxy := modelcomp.NewDynamicModelProxy(modelFactory)

	mcpFactory := toolscomp.NewMCPToolsFactory(cfg.AI, mcpSrv)
	builtinFactory, err := builtintools.NewBuiltinToolsFactory(
		obsDeps.DatasourceSrv,
		obsDeps.DashboardSrv,
		obsDeps.DatasourceMgr,
		obsDeps.AlertRuleSrv,
	)
	if err != nil {
		panic(fmt.Sprintf("ai: failed to build builtin tools: %v", err))
	}
	composite := toolscomp.NewCompositeToolsFactory(mcpFactory, builtinFactory)
	skillRegistry := skillscomp.NewSkillRegistry(db, composite)

	// Create MemoryStore factory and instance
	memoryStoreFactory := memorycomp.NewMemoryStoreFactory(db, convSrv, cfg.AI)
	memoryStore := memoryStoreFactory.GetDefaultStore()

	agent, err := agentcomp.New(cfg.AI, convSrv, modelFactory, skillRegistry, proxy, memoryStore)
	if err != nil {
		panic(fmt.Sprintf("ai: failed to compile react graph: %v", err))
	}

	// Register the Eino global callback that annotates each component with org/user IDs.
	telemetry.Register()

	deps := &agenthandler.Deps{
		Cfg:         cfg.AI,
		Agent:       agent,
		ModelSrv:    modelSrv,
		MCPSrv:      mcpSrv,
		ConvSrv:     convSrv,
		FeedbackSrv: feedbackSrv,
	}
	mod = &aiModule{
		chat:     agenthandler.NewChatHandler(deps),
		model:    agenthandler.NewModelHandler(deps),
		mcp:      agenthandler.NewMCPHandler(deps),
		feedback: agenthandler.NewFeedbackHandler(deps),
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
	chat := mod.chat
	model := mod.model
	mcp := mod.mcp
	fb := mod.feedback

	// Viewer-level routes: any authenticated user may invoke the agent.
	router.POST("/ai/chat",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, chat.Chat)...)
	router.POST("/ai/conversations/:uid/chat",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, chat.Chat)...)
	router.GET("/ai/conversations",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, chat.ListConversations)...)
	router.DELETE("/ai/conversations/:uid",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, chat.DeleteConversation)...)
	router.PUT("/ai/conversations/:uid/title",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, chat.UpdateConversationTitle)...)
	router.GET("/ai/conversations/:uid/messages",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, chat.GetConversationMessages)...)

	// Feedback — viewer-level: any authenticated conversation owner may submit or read feedback.
	router.PUT("/ai/conversations/:uid/messages/:messageUID/feedback",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Write, fb.UpsertFeedback)...)
	router.GET("/ai/conversations/:uid/messages/:messageUID/feedback",
		middleware.Authorize(deps, accesscontrol.ViewerAccessResource, accesscontrol.Read, fb.GetFeedback)...)

	// Admin-level routes: only org admins may configure the agent.

	// Model CRUD — /ai/models/default must be registered before /ai/models/:uid
	// so that Gin's exact-match rule picks up the literal "default" segment first.
	router.GET("/ai/models",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, model.ListModels)...)
	router.POST("/ai/models",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, model.CreateModel)...)
	router.GET("/ai/models/default",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, model.GetDefaultModel)...)
	router.GET("/ai/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, model.GetModel)...)
	router.PUT("/ai/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, model.UpdateModel)...)
	router.DELETE("/ai/models/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, model.DeleteModel)...)
	router.PUT("/ai/models/:uid/default",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, model.SetDefaultModel)...)

	// Routing config
	router.GET("/ai/settings/model-routing",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, model.GetOrgModelConfig)...)
	router.PUT("/ai/settings/model-routing",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, model.SaveOrgModelConfig)...)
	router.GET("/ai/settings/mcp",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Read, mcp.ListMCPServers)...)
	router.POST("/ai/settings/mcp",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, mcp.CreateMCPServer)...)
	router.DELETE("/ai/settings/mcp/:uid",
		middleware.Authorize(deps, accesscontrol.AdminAccessResource, accesscontrol.Write, mcp.DeleteMCPServer)...)
}
