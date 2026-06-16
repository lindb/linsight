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

package template

import (
	"sync"
	"time"
)

// OrgTemplateLoader is a function that loads and merges all notification templates for an org.
// It is called on cache miss and must return the concatenated template content.
type OrgTemplateLoader func(orgID int64) (string, error)

// orgTemplateCacheEntry holds a cached merged template string with its TTL expiry.
type orgTemplateCacheEntry struct {
	merged    string
	expiresAt time.Time
}

// RenderingEngine wraps a Renderer with a per-org TTL cache for org template loading.
// It is the single entry point for rendering notification templates in both the
// alert dispatcher and the HTTP preview handler.
type RenderingEngine struct {
	loadFn   OrgTemplateLoader
	ttl      time.Duration
	mu       sync.RWMutex
	cache    map[int64]*orgTemplateCacheEntry
	renderer Renderer
}

// NewRenderingEngine creates a RenderingEngine.
// loadFn is called when the cache is empty or expired for a given orgID; it must not be nil.
// ttl controls how long the merged template string is cached before a reload.
// A ttl of 0 disables caching (every Render call reloads from the loader).
// The cache is never actively evicted beyond TTL expiry — entries accumulate until the process restarts.
func NewRenderingEngine(loadFn OrgTemplateLoader, ttl time.Duration) *RenderingEngine {
	if loadFn == nil {
		panic("template: NewRenderingEngine: loadFn must not be nil")
	}
	return &RenderingEngine{
		loadFn:   loadFn,
		ttl:      ttl,
		cache:    make(map[int64]*orgTemplateCacheEntry),
		renderer: NewRenderer(),
	}
}

// GetMergedOrgTemplates returns the merged org template content for the given org.
// Results are cached for the configured TTL to avoid repeated DB queries on every alert dispatch.
// Uses a double-checked locking pattern: concurrent cache misses load the template exactly once.
func (e *RenderingEngine) GetMergedOrgTemplates(orgID int64) (string, error) {
	// Fast path: read-lock is sufficient for concurrent readers on a warm cache.
	e.mu.RLock()
	if entry, ok := e.cache[orgID]; ok && time.Now().Before(entry.expiresAt) {
		merged := entry.merged
		e.mu.RUnlock()
		return merged, nil
	}
	e.mu.RUnlock()

	// Slow path: acquire exclusive lock and re-check to ensure only one goroutine
	// calls loadFn when multiple callers race on the same cache miss.
	e.mu.Lock()
	defer e.mu.Unlock()
	if entry, ok := e.cache[orgID]; ok && time.Now().Before(entry.expiresAt) {
		// Another goroutine populated the cache while we were waiting for the lock.
		return entry.merged, nil
	}

	merged, err := e.loadFn(orgID)
	if err != nil {
		return "", err
	}
	e.cache[orgID] = &orgTemplateCacheEntry{
		merged:    merged,
		expiresAt: time.Now().Add(e.ttl),
	}
	return merged, nil
}

// Invalidate removes the cached org templates for the given org so that the next
// render call reloads fresh content from the DB.
// Call this after any Create/Update/Delete operation on notification templates.
func (e *RenderingEngine) Invalidate(orgID int64) {
	e.mu.Lock()
	delete(e.cache, orgID)
	e.mu.Unlock()
}

// Render merges the org-level {{define}} blocks with tmplStr and executes the result.
// The org templates are prepended so their named blocks are registered before the
// root template body in tmplStr runs — identical to the alert dispatcher pipeline.
func (e *RenderingEngine) Render(orgID int64, tmplStr string, data *NotificationData) (string, error) {
	merged, err := e.GetMergedOrgTemplates(orgID)
	if err != nil {
		return "", err
	}
	return renderWithOrgBlocks(merged, tmplStr, data)
}

// RenderNamed loads org templates and executes the named {{define}} block identified by name.
func (e *RenderingEngine) RenderNamed(orgID int64, name string, data *NotificationData) (string, error) {
	merged, err := e.GetMergedOrgTemplates(orgID)
	if err != nil {
		return "", err
	}
	return e.renderer.RenderNamed(merged, name, data)
}
