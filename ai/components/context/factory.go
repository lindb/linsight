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

package context

import (
	"context"
	"fmt"

	"github.com/lindb/common/pkg/logger"

	agentconfig "github.com/lindb/linsight/ai/config"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

var factoryLog = logger.GetLogger("agent", "context_provider_factory")

// ContextProviderFactory creates ContextProvider instances based on configuration.
type ContextProviderFactory interface {
	// CreateContextProvider creates a ContextProvider instance for the specified type.
	CreateContextProvider(ctx context.Context, providerType ProviderType) (ContextProvider, error)

	// GetDefaultProvider returns the default ContextProvider instance.
	GetDefaultProvider() ContextProvider

	// GetProviderByConfig returns a ContextProvider instance based on configuration.
	GetProviderByConfig(cfg *agentconfig.Config) (ContextProvider, error)
}

// contextProviderFactory implements ContextProviderFactory.
type contextProviderFactory struct {
	db              dbpkg.DB
	convSrv         ConversationProvider
	defaultProvider ContextProvider
	config          *agentconfig.Config
}

// NewContextProviderFactory creates a new ContextProviderFactory instance.
func NewContextProviderFactory(db dbpkg.DB, convSrv ConversationProvider, cfg *agentconfig.Config) ContextProviderFactory {
	factory := &contextProviderFactory{
		db:      db,
		convSrv: convSrv,
		config:  cfg,
	}

	// Initialize default provider
	factory.defaultProvider = factory.createDefaultProvider()

	return factory
}

// CreateContextProvider creates a ContextProvider instance based on provider type.
func (f *contextProviderFactory) CreateContextProvider(ctx context.Context, providerType ProviderType) (ContextProvider, error) {
	switch providerType {
	case ProviderTypeDB:
		return f.createDBContextProvider(), nil
	case ProviderTypeRedis:
		return nil, fmt.Errorf("RedisContextProvider not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetDefaultProvider returns the default ContextProvider instance.
func (f *contextProviderFactory) GetDefaultProvider() ContextProvider {
	return f.defaultProvider
}

// GetProviderByConfig returns a ContextProvider instance based on configuration.
func (f *contextProviderFactory) GetProviderByConfig(cfg *agentconfig.Config) (ContextProvider, error) {
	if cfg == nil {
		return f.defaultProvider, nil
	}

	// TODO: Read provider type from configuration when config structure is extended.
	// For now, always return the default DB provider.
	return f.defaultProvider, nil
}

// createDefaultProvider creates the default ContextProvider instance.
func (f *contextProviderFactory) createDefaultProvider() ContextProvider {
	return f.createDBContextProvider()
}

// createDBContextProvider creates a DB-backed ContextProvider instance.
func (f *contextProviderFactory) createDBContextProvider() ContextProvider {
	factoryLog.Info("creating DB-backed ContextProvider instance")
	return NewDBContextProvider(f.db, f.convSrv, f.config.ContextSize)
}
