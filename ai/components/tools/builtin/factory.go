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

// Package builtin provides the built-in observability tools for the LinSight
// AI agent: linsight_list_datasources, linsight_datasource_query, and
// linsight_search_dashboards.
package builtin

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

var toolsLog = logger.GetLogger("agent", "tools")

// truncateStr returns at most maxRunes runes from s, appending "…" when truncated.
func truncateStr(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}

// datasourceReader is the minimal DatasourceService surface that builtin tools require.
// Defined locally to avoid an import cycle with the service package.
type datasourceReader interface {
	GetDatasources(ctx context.Context) ([]model.Datasource, error)
	GetDatasourceByUID(ctx context.Context, uid string) (*model.Datasource, error)
}

// dashboardSearcher is the minimal DashboardService surface that builtin tools require.
type dashboardSearcher interface {
	SearchDashboards(ctx context.Context, req *model.SearchDashboardRequest) (rs []model.Dashboard, total int64, err error)
}

// pluginManager is the minimal datasource.Manager surface that builtin tools require.
type pluginManager interface {
	GetPlugin(datasource *model.Datasource) (plugin.DatasourcePlugin, error)
}

// alertRuleCreator is the minimal AlertRuleService surface that the create-alert-rule
// tool requires. Defined locally to avoid an import cycle with the service package.
type alertRuleCreator interface {
	CreateAlertRule(ctx context.Context, req *model.CreateAlertRuleRequest) (string, error)
}

// BuiltinToolsFactory provides the built-in observability tools.
// Unlike MCPToolsFactory the tool set is the same for all orgs, so the
// tools slice is built once at construction time.
type BuiltinToolsFactory struct {
	tools []tool.BaseTool
}

// NewBuiltinToolsFactory constructs all builtin tools by iterating the registry
// populated by each tool file's init() function.
// Returns an error only if Eino fails to infer a tool's JSON schema, which
// would indicate a programming error in the input struct definition.
func NewBuiltinToolsFactory(
	datasourceSrv datasourceReader,
	dashboardSrv dashboardSearcher,
	datasourceMgr pluginManager,
	alertRuleSrv alertRuleCreator,
) (*BuiltinToolsFactory, error) {
	deps := BuiltinDeps{
		DatasourceSrv: datasourceSrv,
		DashboardSrv:  dashboardSrv,
		DatasourceMgr: datasourceMgr,
		AlertRuleSrv:  alertRuleSrv,
	}
	tools := make([]tool.BaseTool, 0, len(builtinRegistry))
	for _, ctor := range builtinRegistry {
		t, err := ctor(deps)
		if err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return &BuiltinToolsFactory{tools: tools}, nil
}

// GetToolsForOrg returns all built-in tools; built-in tools are org-agnostic.
// The signature satisfies the ToolsProvider interface in the parent tools package.
func (f *BuiltinToolsFactory) GetToolsForOrg(_ context.Context, _ int64) ([]tool.BaseTool, error) {
	return f.tools, nil
}
