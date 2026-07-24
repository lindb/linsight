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
	"context"
	"errors"
)

var ErrNotFound = errors.New("integration not found")

//go:generate mockgen -destination=./mocks/mock_catalog_service.go -package=mocks github.com/lindb/linsight/integrations CatalogService

// CatalogService provides access to integration metadata.
type CatalogService interface {
	ListCategories(ctx context.Context) ([]CategoryCount, error)
	SearchIntegrations(ctx context.Context, category, q string) ([]*IntegrationSummary, error)
	GetIntegration(ctx context.Context, id string) (*Integration, error)
}

// catalogService implements CatalogService.
type catalogService struct {
	store *Store
}

// NewCatalogService creates a new CatalogService.
func NewCatalogService(store *Store) CatalogService {
	return &catalogService{store: store}
}

// noopCatalogService is used when no integrations directory is configured.
type noopCatalogService struct{}

func (noopCatalogService) ListCategories(_ context.Context) ([]CategoryCount, error) {
	return []CategoryCount{}, nil
}

func (noopCatalogService) SearchIntegrations(_ context.Context, _, _ string) ([]*IntegrationSummary, error) {
	return []*IntegrationSummary{}, nil
}

func (noopCatalogService) GetIntegration(_ context.Context, _ string) (*Integration, error) {
	return nil, ErrNotFound
}

// ListCategories lists all available categories and their counts.
func (s *catalogService) ListCategories(ctx context.Context) ([]CategoryCount, error) {
	return s.store.ListCategories()
}

// SearchIntegrations searches for integrations by category and query string.
func (s *catalogService) SearchIntegrations(ctx context.Context, category, q string) ([]*IntegrationSummary, error) {
	return s.store.Search(category, q)
}

// GetIntegration retrieves a specific integration by ID.
func (s *catalogService) GetIntegration(ctx context.Context, id string) (*Integration, error) {
	integ, err := s.store.Get(id)
	if err != nil {
		if errors.Is(err, ErrStoreNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return integ, nil
}
