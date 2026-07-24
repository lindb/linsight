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

package multiagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/lindb/common/pkg/logger"

	agentmodel "github.com/lindb/linsight/ai/model"
	"github.com/lindb/linsight/pkg/markdown"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

// runtimeSpecialist is the cache-resident form of a Specialist row: the fields parsed once
// from its agent.md Content at cache-load time, so the request hot-path performs no parsing.
// Model and Tools come from the YAML frontmatter and are reserved for future per-specialist
// model selection / tool restriction; they are not yet enforced at runtime.
type runtimeSpecialist struct {
	Name         string
	Description  string // frontmatter description → handoff tool Desc / routing hint
	SystemPrompt string // frontmatter body → the specialist's isolated system prompt
	Model        string // frontmatter model (reserved; not yet wired)
	Tools        string // frontmatter tools (reserved; not yet wired)
}

// specialistFrontmatter mirrors the optional YAML frontmatter in a Specialist's Content,
// following the Claude Code agent.md convention.
type specialistFrontmatter struct {
	Description string `yaml:"description"`
	Model       string `yaml:"model"`
	Tools       string `yaml:"tools"`
}

// DBBackend is the dynamic specialist registry: an in-memory, RWMutex-guarded cache of
// enabled Specialists, refreshed from the DB by a background ticker (mirroring
// skills.DBBackend). It implements tools.ToolsProvider so the compiled agent graph gains
// handoff_to_<name> tools with no graph changes — this is the "star topology" hot-plug:
// specialists are registered in the DB and surface as tools bound to the Host.
type DBBackend struct {
	db              dbpkg.DB
	proxy           specialistGenerator
	refreshInterval time.Duration

	mu        sync.RWMutex
	globalMap map[string]*runtimeSpecialist
	orgMaps   map[int64]map[string]*runtimeSpecialist
}

// NewDBBackend creates a DBBackend. Call RefreshCache once before serving and optionally
// StartAutoRefresh for periodic reloads. proxy is used by each handoff tool to run its
// specialist against the caller org's model (*modelcomp.DynamicModelProxy satisfies it).
func NewDBBackend(db dbpkg.DB, proxy specialistGenerator, refreshInterval time.Duration) *DBBackend {
	return &DBBackend{
		db:              db,
		proxy:           proxy,
		refreshInterval: refreshInterval,
		globalMap:       make(map[string]*runtimeSpecialist),
		orgMaps:         make(map[int64]map[string]*runtimeSpecialist),
	}
}

// StartAutoRefresh launches a background goroutine that reloads the cache every
// refreshInterval until ctx is canceled. A non-positive interval disables the ticker.
func (b *DBBackend) StartAutoRefresh(ctx context.Context) {
	if b.refreshInterval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(b.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := b.RefreshCache(ctx); err != nil {
					handoffLog.Warn("multiagent: periodic cache refresh failed", logger.Error(err))
				}
			}
		}
	}()
}

// RefreshCache reloads all enabled specialists and atomically swaps the in-memory cache.
// Rows with an invalid name are skipped (with a warning) rather than failing the reload,
// since an unsafe name would produce an invalid tool the LLM cannot call.
func (b *DBBackend) RefreshCache(ctx context.Context) error {
	var rows []*agentmodel.Specialist
	if err := b.db.RawDB().WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("name asc").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("multiagent: load from db: %w", err)
	}

	globalMap := make(map[string]*runtimeSpecialist)
	orgMaps := make(map[int64]map[string]*runtimeSpecialist)
	for _, row := range rows {
		if err := agentmodel.ValidateName(row.Name); err != nil {
			handoffLog.Warn("multiagent: skipping specialist with invalid name",
				logger.String("name", row.Name), logger.Error(err))
			continue
		}
		rs, err := parseSpecialist(row.Name, row.Content)
		if err != nil {
			handoffLog.Warn("multiagent: skipping specialist with malformed frontmatter",
				logger.String("name", row.Name), logger.Error(err))
			continue
		}
		if row.IsGlobal {
			globalMap[rs.Name] = rs
		} else {
			m, ok := orgMaps[row.OrgID]
			if !ok {
				m = make(map[string]*runtimeSpecialist)
				orgMaps[row.OrgID] = m
			}
			m[rs.Name] = rs
		}
	}

	b.mu.Lock()
	b.globalMap = globalMap
	b.orgMaps = orgMaps
	b.mu.Unlock()

	handoffLog.Info("multiagent: cache refreshed",
		logger.Int("global", len(globalMap)), logger.Int("orgs", len(orgMaps)))
	return nil
}

// GetToolsForOrg implements tools.ToolsProvider: it returns one handoff tool per specialist
// visible to orgID (global specialists plus that org's own). Global specialists win on a
// name collision. The result is safe to bind to the Host as regular Eino tools.
func (b *DBBackend) GetToolsForOrg(_ context.Context, orgID int64) ([]tool.BaseTool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// De-duplicate by name so an org cannot shadow a global specialist's tool name.
	specs := make(map[string]*runtimeSpecialist, len(b.globalMap)+len(b.orgMaps[orgID]))
	for name, rs := range b.orgMaps[orgID] {
		specs[name] = rs
	}
	for name, rs := range b.globalMap {
		specs[name] = rs
	}

	tools := make([]tool.BaseTool, 0, len(specs))
	for _, rs := range specs {
		tools = append(tools, newHandoffTool(rs, b.proxy))
	}
	return tools, nil
}

// parseSpecialist splits the agent.md Content into its optional YAML frontmatter and body,
// mapping frontmatter → Description/Model/Tools and the body → SystemPrompt. When no
// frontmatter is present the whole Content is the system prompt (Description empty). A
// present-but-malformed frontmatter block returns an error so the caller can skip the row.
func parseSpecialist(name, content string) (*runtimeSpecialist, error) {
	rs := &runtimeSpecialist{Name: name}

	meta, body, err := markdown.ParseFrontmatter[specialistFrontmatter](content)
	if err != nil {
		return nil, err
	}

	rs.SystemPrompt = strings.TrimSpace(body)
	rs.Description = strings.TrimSpace(meta.Description)
	rs.Model = strings.TrimSpace(meta.Model)
	rs.Tools = strings.TrimSpace(meta.Tools)
	return rs, nil
}
