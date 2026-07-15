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

package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lindb/common/pkg/logger"

	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

var log = logger.GetLogger("agent", "memory")

// Backend loads memories for an org and renders them into the system prompt.
type Backend interface {
	// Render formats all enabled memories for the org in ctx as a structured text
	// block suitable for injection into the system prompt. Returns an empty string
	// when the org has no memories or the user is not present in ctx.
	Render(ctx context.Context) (string, error)
	// RefreshCache reloads all enabled memories from DB and atomically swaps the cache.
	RefreshCache(ctx context.Context) error
	// StartAutoRefresh launches a background goroutine that reloads the cache every
	// refreshInterval until ctx is canceled. A non-positive interval disables the ticker.
	StartAutoRefresh(ctx context.Context)
}

// runtimeMemory is the cache-resident form of a Memory row.
type runtimeMemory struct {
	UID      string
	Content  string
	Category string
}

// DBMemoryBackend implements Backend with an in-memory, RWMutex-guarded cache.
// Memories are bucketed by org_id for O(1) per-org lookup. The cache is refreshed
// via a background ticker so that DB writes propagate automatically.
type DBMemoryBackend struct {
	db              dbpkg.DB
	refreshInterval time.Duration

	mu      sync.RWMutex
	orgMems map[int64][]*runtimeMemory
}

// NewDBMemoryBackend creates a DBMemoryBackend. Call RefreshCache once before serving
// and optionally StartAutoRefresh for periodic reloads.
func NewDBMemoryBackend(db dbpkg.DB, refreshInterval time.Duration) *DBMemoryBackend {
	return &DBMemoryBackend{
		db:              db,
		refreshInterval: refreshInterval,
		orgMems:         make(map[int64][]*runtimeMemory),
	}
}

// StartAutoRefresh launches a background goroutine that reloads the cache every
// refreshInterval until ctx is canceled. A non-positive interval disables the ticker.
func (b *DBMemoryBackend) StartAutoRefresh(ctx context.Context) {
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
					log.Warn("memory: periodic cache refresh failed", logger.Error(err))
				}
			}
		}
	}()
}

// RefreshCache reloads all enabled memories and atomically swaps the in-memory cache.
func (b *DBMemoryBackend) RefreshCache(ctx context.Context) error {
	var rows []*agentmodel.Memory
	if err := b.db.RawDB().WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("category asc, id asc").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("memory: load from db: %w", err)
	}

	orgMems := make(map[int64][]*runtimeMemory)
	for _, row := range rows {
		rm := &runtimeMemory{
			UID:      row.UID,
			Content:  row.Content,
			Category: row.Category,
		}
		orgMems[row.OrgID] = append(orgMems[row.OrgID], rm)
	}

	b.mu.Lock()
	b.orgMems = orgMems
	b.mu.Unlock()

	total := len(rows)
	log.Info("memory: cache refreshed", logger.Int("total", total), logger.Int("orgs", len(orgMems)))
	return nil
}

// Render formats all enabled memories for the org in ctx into a structured text block.
// Each memory entry is prefixed with its short UID so the LLM can reference it in
// save/delete tool calls. Returns an empty string when the org has no memories.
func (b *DBMemoryBackend) Render(ctx context.Context) (string, error) {
	orgID, err := getOrgID(ctx)
	if err != nil {
		// No signed user — return empty, don't leak cross-org data.
		return "", nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	entries := b.orgMems[orgID]
	if len(entries) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(memoryBlockHeader)
	for i, m := range entries {
		if m.Category != "" {
			fmt.Fprintf(&sb, "%d. [%s] (%s) %s\n", i+1, m.UID, m.Category, m.Content)
		} else {
			fmt.Fprintf(&sb, "%d. [%s] %s\n", i+1, m.UID, m.Content)
		}
	}
	return sb.String(), nil
}

// getOrgID extracts the org ID from ctx. Returns an error when the user is not available.
func getOrgID(ctx context.Context) (int64, error) {
	u := util.GetUser(ctx)
	if u == nil || u.Org == nil {
		return 0, fmt.Errorf("memory: no signed user in context")
	}
	return u.Org.ID, nil
}

// memoryBlockHeader is the header text that precedes the injected memory entries
// inside the system prompt. It instructs the LLM that these are persistent user facts.
const memoryBlockHeader = `

# User Long-Term Memory
The following facts about the user have been persisted across sessions. Treat them as
authoritative context. When the user asks to remember, update, or forget something,
use the save_memory / delete_memory tools:

`
