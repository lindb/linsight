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

package tools

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/lindb/common/pkg/logger"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	agentconfig "github.com/lindb/linsight/ai/config"
	agentmodel "github.com/lindb/linsight/ai/model"
)

var log = logger.GetLogger("agent", "tools")

// mcpSettingsReader is the minimal interface that MCPToolsFactory needs from the
// settings layer. Defined locally to avoid an import cycle with agent/service.
type mcpSettingsReader interface {
	ListMCPServers(ctx context.Context) ([]*agentmodel.MCP, error)
}

// toolsCacheEntry holds the fetched tool list and the hash it was built from.
type toolsCacheEntry struct {
	hash  string
	tools []tool.BaseTool
}

// MCPToolsFactory builds and caches per-org MCP tool lists.
// Tools are rebuilt only when the org's server configuration changes.
type MCPToolsFactory struct {
	cfg         *agentconfig.Config
	settingsSrv mcpSettingsReader
	// cache maps orgID (int64) → *toolsCacheEntry
	cache sync.Map
}

// NewMCPToolsFactory creates an MCPToolsFactory backed by settingsSrv for per-org MCP configs.
func NewMCPToolsFactory(cfg *agentconfig.Config, settingsSrv mcpSettingsReader) *MCPToolsFactory {
	return &MCPToolsFactory{cfg: cfg, settingsSrv: settingsSrv}
}

// GetToolsForOrg returns the combined list of MCP tools available to orgID.
// Results are cached per org and invalidated when any server's ConfigHash changes.
func (f *MCPToolsFactory) GetToolsForOrg(ctx context.Context, orgID int64) ([]tool.BaseTool, error) {
	servers, err := f.settingsSrv.ListMCPServers(ctx)
	if err != nil {
		return nil, err
	}

	// Compute a combined hash of all enabled server configs.
	combinedHash := f.combinedHash(servers)

	// Fast path: return cached tools when hash matches.
	if v, ok := f.cache.Load(orgID); ok {
		entry := v.(*toolsCacheEntry)
		if entry.hash == combinedHash {
			return entry.tools, nil
		}
	}

	// Slow path: rebuild tool list from scratch.
	tools, err := f.buildTools(ctx, servers)
	if err != nil {
		return nil, err
	}
	f.cache.Store(orgID, &toolsCacheEntry{hash: combinedHash, tools: tools})
	return tools, nil
}

// Invalidate evicts the cached tools for orgID, forcing a rebuild on the next request.
func (f *MCPToolsFactory) Invalidate(orgID int64) {
	f.cache.Delete(orgID)
}

// buildTools connects to each enabled MCP server and fetches its tool list.
// Failures are fail-open: a server that cannot connect is logged and skipped.
func (f *MCPToolsFactory) buildTools(ctx context.Context, servers []*agentmodel.MCP) ([]tool.BaseTool, error) {
	var result []tool.BaseTool
	for _, srv := range servers {
		if !srv.Enabled {
			continue
		}
		tools, err := f.loadToolsFromServer(ctx, srv)
		if err != nil {
			// Fail-open: log and skip unavailable servers rather than failing the whole request.
			log.Warn("skipping unavailable MCP server", logger.String("name", srv.Name), logger.Error(err))
			continue
		}
		result = append(result, tools...)
	}
	return result, nil
}

// loadToolsFromServer connects to a single MCP server and returns its tools.
func (f *MCPToolsFactory) loadToolsFromServer(ctx context.Context, srv *agentmodel.MCP) ([]tool.BaseTool, error) {
	// Apply per-tool-call timeout to the connection + list phase.
	timeout := time.Duration(f.cfg.ToolTimeout)
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cli mcpclient.MCPClient
	var err error

	switch srv.Type {
	case agentmodel.MCPServerTypeSSE:
		cli, err = mcpclient.NewSSEMCPClient(srv.URL)
		if err != nil {
			return nil, err
		}
	case agentmodel.MCPServerTypeStdio:
		args := splitArgs(srv.Args)
		cli, err = mcpclient.NewStdioMCPClient(srv.Command, nil, args...)
		if err != nil {
			return nil, err
		}
	default:
		log.Warn("unknown MCP server type, skipping", logger.String("type", string(srv.Type)))
		return nil, nil
	}

	// Initialize the MCP session before listing tools.
	if _, err = cli.Initialize(toolCtx, mcp.InitializeRequest{}); err != nil {
		return nil, err
	}

	return einomcp.GetTools(toolCtx, &einomcp.Config{Cli: cli})
}

// combinedHash builds a single fingerprint covering all server config hashes.
// The order is deterministic because ListMCPServers returns rows ordered by ID.
func (f *MCPToolsFactory) combinedHash(servers []*agentmodel.MCP) string {
	parts := make([]string, 0, len(servers))
	for _, s := range servers {
		parts = append(parts, s.ConfigHash())
	}
	return strings.Join(parts, "|")
}

// splitArgs splits a whitespace-separated argument string into a slice.
// Handles multiple spaces between tokens; returns nil for an empty string.
func splitArgs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	return strings.Fields(raw)
}
