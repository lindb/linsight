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

package skills

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk/middlewares/skill"
	"github.com/lindb/common/pkg/logger"
	"gopkg.in/yaml.v3"

	agentmodel "github.com/lindb/linsight/ai/model"
	aiutil "github.com/lindb/linsight/ai/util"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

var log = logger.GetLogger("agent", "skills")

// Backend loads skills for an org. org scoping is handled internally via ctx.
type Backend interface {
	// List returns frontmatter for all skills visible to the org in ctx.
	List(ctx context.Context) ([]skill.FrontMatter, error)
	// Get returns a single skill by name, scoped to the org in ctx.
	Get(ctx context.Context, name string) (skill.Skill, error)
}

// runtimeSkill is the cache-resident form of a Skill. YAML frontmatter is parsed
// once at cache-load time so the request hot-path performs no parsing.
type runtimeSkill struct {
	Name        string
	Description string
	Content     string
	ContextMode agentmodel.SkillContextMode
	Agent       string
}

// skillFrontmatter mirrors the optional YAML frontmatter in a Skill's Content.
type skillFrontmatter struct {
	Description string `yaml:"description"`
	ContextMode string `yaml:"context_mode"`
	Agent       string `yaml:"agent"`
}

// DBBackend implements Backend with an in-memory, RWMutex-guarded cache.
// Global skills (is_global=true) apply to every org; per-org skills are
// bucketed by org_id. The cache is refreshed via a background ticker so
// that DB writes propagate automatically without per-request DB queries.
type DBBackend struct {
	db              dbpkg.DB
	refreshInterval time.Duration

	mu        sync.RWMutex
	globalMap map[string]*runtimeSkill
	orgMaps   map[int64]map[string]*runtimeSkill
}

// NewDBBackend creates a DBBackend. Call RefreshCache once before serving and
// optionally StartAutoRefresh for periodic reloads.
func NewDBBackend(db dbpkg.DB, refreshInterval time.Duration) *DBBackend {
	return &DBBackend{
		db:              db,
		refreshInterval: refreshInterval,
		globalMap:       make(map[string]*runtimeSkill),
		orgMaps:         make(map[int64]map[string]*runtimeSkill),
	}
}

// StartAutoRefresh launches a background goroutine that reloads the cache every
// refreshInterval until ctx is cancelled. A non-positive interval disables the ticker.
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
					log.Warn("skills: periodic cache refresh failed", logger.Error(err))
				}
			}
		}
	}()
}

// RefreshCache reloads all enabled skills and atomically swaps the in-memory cache.
// Map-based storage for O(1) lookup on the request hot-path.
// A skill with malformed frontmatter is skipped (with a warning) rather than failing
// the whole reload.
func (b *DBBackend) RefreshCache(ctx context.Context) error {
	var rows []*agentmodel.Skill
	if err := b.db.RawDB().WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("name asc").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("skills: load from db: %w", err)
	}

	globalMap := make(map[string]*runtimeSkill)
	orgMaps := make(map[int64]map[string]*runtimeSkill)
	for _, row := range rows {
		rs, err := parseFrontmatter(row.Name, row.Content)
		if err != nil {
			log.Warn("skills: skipping skill with malformed frontmatter",
				logger.String("name", row.Name), logger.Error(err))
			continue
		}
		if row.IsGlobal {
			globalMap[rs.Name] = rs
		} else {
			m, ok := orgMaps[row.OrgID]
			if !ok {
				m = make(map[string]*runtimeSkill)
				orgMaps[row.OrgID] = m
			}
			m[rs.Name] = rs
		}
	}

	b.mu.Lock()
	b.globalMap = globalMap
	b.orgMaps = orgMaps
	b.mu.Unlock()

	log.Info("skills: cache refreshed", logger.Int("global", len(globalMap)), logger.Int("orgs", len(orgMaps)))
	return nil
}

// List returns frontmatter for all skills visible to the org in ctx.
func (b *DBBackend) List(ctx context.Context) ([]skill.FrontMatter, error) {
	orgID, err := getOrgID(ctx)
	if err != nil {
		return nil, err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	total := len(b.globalMap) + len(b.orgMaps[orgID])
	matters := make([]skill.FrontMatter, 0, total)
	for _, rs := range b.globalMap {
		matters = append(matters, skill.FrontMatter{
			Name:        rs.Name,
			Description: rs.Description,
			Context:     skill.ContextMode(rs.ContextMode),
			Agent:       rs.Agent,
		})
	}
	for _, rs := range b.orgMaps[orgID] {
		matters = append(matters, skill.FrontMatter{
			Name:        rs.Name,
			Description: rs.Description,
			Context:     skill.ContextMode(rs.ContextMode),
			Agent:       rs.Agent,
		})
	}
	return matters, nil
}

// Get returns a single skill by name, scoped to the org in ctx.
func (b *DBBackend) Get(ctx context.Context, name string) (skill.Skill, error) {
	orgID, err := getOrgID(ctx)
	if err != nil {
		return skill.Skill{}, err
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	rs := b.lookup(name, orgID)
	if rs == nil {
		return skill.Skill{}, fmt.Errorf("skills: skill %q not found for org %d", name, orgID)
	}

	return skill.Skill{
		FrontMatter: skill.FrontMatter{
			Name:        rs.Name,
			Description: rs.Description,
			Context:     skill.ContextMode(rs.ContextMode),
			Agent:       rs.Agent,
		},
		Content: rs.Content,
	}, nil
}

// lookup searches globalMap and the org map for a skill by name (O(1)).
// Must be called with b.mu held (read or write).
func (b *DBBackend) lookup(name string, orgID int64) *runtimeSkill {
	if rs, ok := b.globalMap[name]; ok {
		return rs
	}
	if m, ok := b.orgMaps[orgID]; ok {
		return m[name]
	}
	return nil
}

// getOrgID extracts the org ID from ctx. Returns an error when the user is not
// available, rather than silently falling back to 0 (which could leak global skills).
func getOrgID(ctx context.Context) (int64, error) {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil {
		return 0, fmt.Errorf("skills: no signed user in context")
	}
	return u.Org.ID, nil
}

// parseFrontmatter splits the optional YAML frontmatter from the Markdown body.
// When no frontmatter is present the whole content is treated as the body (an inline
// skill with no description or agent).
func parseFrontmatter(name string, content string) (*runtimeSkill, error) {
	rs := &runtimeSkill{Name: name}

	fm, body := aiutil.SplitFrontmatter(content)
	rs.Content = body

	if fm == "" {
		return rs, nil
	}

	var meta skillFrontmatter
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	rs.Description = strings.TrimSpace(meta.Description)
	rs.Agent = strings.TrimSpace(meta.Agent)

	switch strings.TrimSpace(meta.ContextMode) {
	case "fork":
		rs.ContextMode = agentmodel.SkillContextModeFork
	case "fork_with_context":
		rs.ContextMode = agentmodel.SkillContextModeForkWithContext
	default:
		rs.ContextMode = agentmodel.SkillContextModeInline
	}

	return rs, nil
}
