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

	"github.com/lindb/common/pkg/logger"

	agentconfig "github.com/lindb/linsight/ai/config"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

var factoryLog = logger.GetLogger("agent", "memory_store_factory")

// MemoryStoreFactory Factory for creating MemoryStore instances based on configuration
type MemoryStoreFactory interface {
	// CreateMemoryStore creates a MemoryStore instance based on the specified type
	CreateMemoryStore(ctx context.Context, storeType StoreType) (MemoryStore, error)

	// GetDefaultStore returns the default MemoryStore instance
	GetDefaultStore() MemoryStore

	// GetStoreByConfig returns a MemoryStore instance based on configuration
	GetStoreByConfig(cfg *agentconfig.Config) (MemoryStore, error)
}

// memoryStoreFactory Implementation of MemoryStoreFactory
type memoryStoreFactory struct {
	db           dbpkg.DB
	convSrv      ConversationProvider
	defaultStore MemoryStore
	config       *agentconfig.Config
}

// NewMemoryStoreFactory creates a new MemoryStoreFactory instance
func NewMemoryStoreFactory(db dbpkg.DB, convSrv ConversationProvider, cfg *agentconfig.Config) MemoryStoreFactory {
	factory := &memoryStoreFactory{
		db:      db,
		convSrv: convSrv,
		config:  cfg,
	}

	// Initialize default store
	factory.defaultStore = factory.createDefaultStore()

	return factory
}

// CreateMemoryStore creates a MemoryStore instance based on store type
func (f *memoryStoreFactory) CreateMemoryStore(ctx context.Context, storeType StoreType) (MemoryStore, error) {
	switch storeType {
	case StoreTypeDB:
		return f.createDBMemoryStore(), nil
	case StoreTypeRedis:
		return nil, fmt.Errorf("RedisMemoryStore not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported store type: %s", storeType)
	}
}

// GetDefaultStore returns the default MemoryStore instance
func (f *memoryStoreFactory) GetDefaultStore() MemoryStore {
	return f.defaultStore
}

// GetStoreByConfig returns a MemoryStore instance based on configuration
func (f *memoryStoreFactory) GetStoreByConfig(cfg *agentconfig.Config) (MemoryStore, error) {
	if cfg == nil {
		return f.defaultStore, nil
	}

	// TODO: Read store type from configuration when config structure is extended
	// For now, always return the default DB store
	return f.defaultStore, nil
}

// createDefaultStore creates the default MemoryStore instance
func (f *memoryStoreFactory) createDefaultStore() MemoryStore {
	return f.createDBMemoryStore()
}

// createDBMemoryStore creates a DBMemoryStore instance
func (f *memoryStoreFactory) createDBMemoryStore() MemoryStore {
	factoryLog.Info("creating DBMemoryStore instance")
	return NewDBMemoryStore(f.db, f.convSrv, f.config.MemorySize)
}
