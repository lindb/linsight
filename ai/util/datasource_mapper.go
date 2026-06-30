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

package util

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/lindb/linsight/model"
)

// DatasourceMapper provides mapping from data source names to UIDs
type DatasourceMapper struct {
	datasourceService DatasourceService
	cache             map[string]string // name -> UID mapping
	cacheMutex        sync.RWMutex
	lastRefresh       time.Time
	cacheTTL          time.Duration
}

// DatasourceService defines the minimal interface for data source operations
type DatasourceService interface {
	GetDatasources(ctx context.Context) ([]model.Datasource, error)
}

// NewDatasourceMapper creates a new data source mapper
func NewDatasourceMapper(service DatasourceService) *DatasourceMapper {
	return &DatasourceMapper{
		datasourceService: service,
		cache:             make(map[string]string),
		cacheTTL:          5 * time.Minute, // Cache for 5 minutes
	}
}

// MapNameToUID maps a data source name to its UID
func (dm *DatasourceMapper) MapNameToUID(ctx context.Context, name string) (string, error) {
	// Normalize the name for matching
	normalizedName := strings.ToLower(strings.TrimSpace(name))

	// Check cache first
	dm.cacheMutex.RLock()
	if uid, found := dm.cache[normalizedName]; found && time.Since(dm.lastRefresh) < dm.cacheTTL {
		dm.cacheMutex.RUnlock()
		return uid, nil
	}
	dm.cacheMutex.RUnlock()

	// Refresh cache if expired
	if err := dm.refreshCache(ctx); err != nil {
		return "", err
	}

	// Check cache again after refresh
	dm.cacheMutex.RLock()
	uid, found := dm.cache[normalizedName]
	dm.cacheMutex.RUnlock()

	if !found {
		return "", nil // Return empty string if not found
	}

	return uid, nil
}

// refreshCache refreshes the data source cache
func (dm *DatasourceMapper) refreshCache(ctx context.Context) error {
	datasources, err := dm.datasourceService.GetDatasources(ctx)
	if err != nil {
		return err
	}

	newCache := make(map[string]string)
	for _, ds := range datasources {
		// Use name as key (normalized to lowercase)
		key := strings.ToLower(strings.TrimSpace(ds.Name))
		newCache[key] = ds.UID

		// Also add UID as key for direct UID lookup
		newCache[strings.ToLower(ds.UID)] = ds.UID
	}

	dm.cacheMutex.Lock()
	dm.cache = newCache
	dm.lastRefresh = time.Now()
	dm.cacheMutex.Unlock()

	return nil
}

// FindBestMatch finds the best matching data source for a given name
func (dm *DatasourceMapper) FindBestMatch(ctx context.Context, name string) (string, error) {
	// Try exact match first
	uid, err := dm.MapNameToUID(ctx, name)
	if err != nil {
		return "", err
	}
	if uid != "" {
		return uid, nil
	}

	// If exact match fails, try partial matching
	dm.cacheMutex.RLock()
	defer dm.cacheMutex.RUnlock()

	normalizedName := strings.ToLower(strings.TrimSpace(name))
	for cachedName, cachedUID := range dm.cache {
		if strings.Contains(cachedName, normalizedName) || strings.Contains(normalizedName, cachedName) {
			return cachedUID, nil
		}
	}

	return "", nil // No match found
}