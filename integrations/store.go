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

package integrations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lindb/linsight/pkg/markdown"
)

var ErrStoreNotFound = errors.New("integration not found")

// sectionMeta holds frontmatter metadata for a markdown section.
type sectionMeta struct {
	Title string `yaml:"title"`
	Order int    `yaml:"order"`
}

// cacheEntry holds an integration plus file mtimes for each markdown file.
type cacheEntry struct {
	integration *Integration
	mdMtimes    map[string]time.Time // filename -> mtime
}

// Store loads and caches integrations from the filesystem with mtime-based invalidation.
type Store struct {
	mu       sync.RWMutex
	dir      string
	cache    map[string]*cacheEntry
	dirIDs   []string // ordered list of integration IDs for iteration
}

// NewStore initializes the store by scanning the given directory.
func NewStore(dir string) (*Store, error) {
	s := &Store{
		dir:    dir,
		cache:  make(map[string]*cacheEntry),
		dirIDs: []string{},
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read integrations directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		s.dirIDs = append(s.dirIDs, id)
		// Lazy load: only load on first access via Get or Search
	}

	return s, nil
}

// loadOrReload checks file mtimes and reloads if any have changed.
func (s *Store) loadOrReload(id string) error {
	dir := filepath.Join(s.dir, id)
	manifestPath := filepath.Join(dir, "manifest.yaml")

	// Get current file stats
	manifestStat, err := os.Stat(manifestPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat manifest for %s: %w", id, err)
	}

	// Scan all .md files in the directory
	mdFiles := make(map[string]os.FileInfo)
	if dirEntries, err := os.ReadDir(dir); err == nil {
		for _, entry := range dirEntries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
				if info, err := entry.Info(); err == nil {
					mdFiles[entry.Name()] = info
				}
			}
		}
	}

	// Build mtimes map for comparison
	currentMtimes := make(map[string]time.Time)
	if manifestStat != nil {
		currentMtimes["manifest.yaml"] = manifestStat.ModTime()
	}
	for name, info := range mdFiles {
		currentMtimes[name] = info.ModTime()
	}

	// Check if cached entry exists and is up-to-date
	if cached, ok := s.cache[id]; ok {
		if mapsEqual(cached.mdMtimes, currentMtimes) {
			return nil // Cache hit
		}
	}

	// Cache miss or stale: reload files
	var manifest Manifest
	if manifestStat != nil {
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("failed to read manifest for %s: %w", id, err)
		}
		// Parse manifest with markdown.ParseFrontmatter (fallback to direct yaml)
		// Since manifest has no frontmatter, unmarshal directly
		err = yaml.Unmarshal(manifestData, &manifest)
		if err != nil {
			return fmt.Errorf("failed to parse manifest for %s: %w", id, err)
		}
	}

	// Parse all markdown sections
	sections := []ContentSection{}
	for _, filename := range sortedKeys(mdFiles) {
		path := filepath.Join(dir, filename)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s for %s: %w", filename, id, err)
		}

		meta, body, err := markdown.ParseFrontmatter[sectionMeta](string(content))
		if err != nil {
			return fmt.Errorf("failed to parse frontmatter in %s/%s: %w", id, filename, err)
		}

		// If no frontmatter, title defaults to filename without .md
		title := meta.Title
		if title == "" {
			title = strings.TrimSuffix(filename, ".md")
		}

		sections = append(sections, ContentSection{
			Title:   title,
			Content: body,
			Order:   meta.Order,
		})
	}

	// Sort sections by order
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	// Extract overview from the first section's content
	var overview string
	if len(sections) > 0 {
		overview = extractOverview(sections[0].Content)
	}

	integration := &Integration{
		IntegrationSummary: IntegrationSummary{
			Manifest: manifest,
			Overview: overview,
		},
		Sections: sections,
	}

	s.cache[id] = &cacheEntry{
		integration: integration,
		mdMtimes:    currentMtimes,
	}

	return nil
}

// ListCategories returns all categories and their counts.
func (s *Store) ListCategories() ([]CategoryCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reload all entries to get latest data
	for _, id := range s.dirIDs {
		_ = s.loadOrReload(id)
	}

	categoryMap := make(map[string]int)
	for _, cached := range s.cache {
		if cached.integration != nil {
			cat := cached.integration.Category
			categoryMap[cat]++
		}
	}

	result := make([]CategoryCount, 0, len(categoryMap))
	for cat, count := range categoryMap {
		result = append(result, CategoryCount{Category: cat, Count: count})
	}

	return result, nil
}

// Search filters integrations by category and query string.
func (s *Store) Search(category, q string) ([]*IntegrationSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reload all entries
	for _, id := range s.dirIDs {
		_ = s.loadOrReload(id)
	}

	q = strings.ToLower(q)
	var result []*IntegrationSummary

	for _, cached := range s.cache {
		if cached.integration == nil {
			continue
		}

		integ := cached.integration
		manifest := &integ.Manifest

		// Filter by category if specified
		if category != "" && manifest.Category != category {
			continue
		}

		// Filter by query string (case-insensitive substring match)
		if q != "" {
			matched := false
			matched = matched || strings.Contains(strings.ToLower(manifest.Name), q)
			matched = matched || strings.Contains(strings.ToLower(manifest.Description), q)
			for _, tag := range manifest.Tags {
				if strings.Contains(strings.ToLower(tag), q) {
					matched = true
					break
				}
			}
			for _, symptom := range manifest.Symptoms {
				if strings.Contains(strings.ToLower(symptom), q) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		result = append(result, &integ.IntegrationSummary)
	}

	return result, nil
}

// Get retrieves a specific integration by ID.
func (s *Store) Get(id string) (*Integration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if id exists in the directory listing
	found := false
	for _, dirID := range s.dirIDs {
		if dirID == id {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrStoreNotFound
	}

	if err := s.loadOrReload(id); err != nil {
		return nil, err
	}

	cached, ok := s.cache[id]
	if !ok || cached.integration == nil {
		return nil, ErrStoreNotFound
	}

	return cached.integration, nil
}

// extractOverview extracts the first ## Overview section from markdown.
func extractOverview(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var in bool
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for ## Overview
		if strings.HasPrefix(trimmed, "## ") && !in {
			if strings.Contains(trimmed, "Overview") {
				in = true
				continue
			}
		}

		if in {
			// Stop at next heading
			if strings.HasPrefix(trimmed, "## ") && !strings.Contains(trimmed, "Overview") {
				break
			}
			if strings.HasPrefix(trimmed, "###") {
				break
			}
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}

	return strings.TrimSpace(strings.Join(result, "\n"))
}

// mapsEqual checks if two mtime maps are equal.
func mapsEqual(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !v.Equal(bv) {
			return false
		}
	}
	return true
}

// sortedKeys returns sorted keys from a map.
func sortedKeys(m map[string]os.FileInfo) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
