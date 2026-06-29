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

// Package tools provides MCP-based and built-in Eino tool implementations
// for the LinSight AI agent module.
package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	toolutils "github.com/cloudwego/eino/components/tool/utils"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

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

// BuiltinToolsFactory provides the three built-in observability tools.
// Unlike MCPToolsFactory the tool set is the same for all orgs, so the
// tools slice is built once at construction time.
type BuiltinToolsFactory struct {
	tools []tool.BaseTool
}

// NewBuiltinToolsFactory constructs and registers the three built-in tools.
// Returns an error only if Eino fails to infer a tool's JSON schema, which
// would indicate a programming error in the input struct definition.
func NewBuiltinToolsFactory(
	datasourceSrv datasourceReader,
	dashboardSrv dashboardSearcher,
	datasourceMgr pluginManager,
) (*BuiltinToolsFactory, error) {
	listDS, err := newListDatasourcesTool(datasourceSrv)
	if err != nil {
		return nil, fmt.Errorf("builtin: linsight_list_datasources: %w", err)
	}
	query, err := newDatasourceQueryTool(datasourceSrv, datasourceMgr)
	if err != nil {
		return nil, fmt.Errorf("builtin: linsight_datasource_query: %w", err)
	}
	searchDash, err := newSearchDashboardsTool(dashboardSrv)
	if err != nil {
		return nil, fmt.Errorf("builtin: linsight_search_dashboards: %w", err)
	}

	return &BuiltinToolsFactory{
		tools: []tool.BaseTool{listDS, query, searchDash},
	}, nil
}

// GetToolsForOrg returns all built-in tools; built-in tools are org-agnostic.
// The signature satisfies the ToolsProvider interface.
func (f *BuiltinToolsFactory) GetToolsForOrg(_ context.Context, _ int64) ([]tool.BaseTool, error) {
	return f.tools, nil
}

// newListDatasourcesTool builds the linsight_list_datasources Eino tool.
func newListDatasourcesTool(srv datasourceReader) (tool.BaseTool, error) {
	return toolutils.InferTool(
		"linsight_list_datasources",
		"List all data sources accessible to the current user. "+
			"Call this first to discover available data sources before querying data.",
		func(ctx context.Context, _ struct{}) ([]datasourceInfo, error) {
			return listDatasources(ctx, srv)
		},
	)
}

// newDatasourceQueryTool builds the linsight_datasource_query Eino tool.
func newDatasourceQueryTool(srv datasourceReader, mgr pluginManager) (tool.BaseTool, error) {
	return toolutils.InferTool(
		"linsight_datasource_query",
		"Execute a data query against a LinSight datasource. "+
			"Supports metrics (LinDB), logs (LinDB), and traces (LinGo). "+
			"Call linsight_list_datasources first to obtain a valid datasource_uid.",
		func(ctx context.Context, input datasourceQueryInput) (*queryResult, error) {
			return queryDatasource(ctx, srv, mgr, input)
		},
	)
}

// newSearchDashboardsTool builds the linsight_search_dashboards Eino tool.
func newSearchDashboardsTool(srv dashboardSearcher) (tool.BaseTool, error) {
	return toolutils.InferTool(
		"linsight_search_dashboards",
		"Search for dashboards by title or tags. "+
			"Returns a list of matching dashboards with their UIDs and titles.",
		func(ctx context.Context, input searchDashboardsInput) ([]dashboardInfo, error) {
			return searchDashboards(ctx, srv, input)
		},
	)
}
