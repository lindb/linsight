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
	"os"
	"path/filepath"
	"testing"
)

func TestStore(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test integration directory
	integDir := filepath.Join(tmpDir, "test-integration")
	if err := os.Mkdir(integDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Write test manifest.yaml
	manifestContent := `id: test-integration
name: Test Integration
version: 1.0.0
category: observability
description: A test integration for unit testing
tags:
  - test
  - example
symptoms:
  - High latency
  - Service down
maintainers:
  - "@testuser"
homepage: "https://example.com"`

	if err := os.WriteFile(filepath.Join(integDir, "manifest.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Write test README.md with frontmatter
	readmeContent := `---
title: "Setup Guide"
order: 1
---
# Test Integration

## Overview
This is a test integration for demonstrating the knowledge base API.

## Configuration & Setup
Follow these steps to set up the test integration.

## Data Collected
Test data collection setup.`

	if err := os.WriteFile(filepath.Join(integDir, "README.md"), []byte(readmeContent), 0644); err != nil {
		t.Fatalf("failed to write README: %v", err)
	}

	// Write test monitors.md with frontmatter
	monitorsContent := `---
title: "Monitors"
order: 2
---
# Test Monitors

## Monitor List

### High Latency
- **Condition**: latency > 1000ms
- **Severity**: Warning
- **Response**: Check service performance`

	if err := os.WriteFile(filepath.Join(integDir, "monitors.md"), []byte(monitorsContent), 0644); err != nil {
		t.Fatalf("failed to write monitors: %v", err)
	}

	// Create store
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Test Get
	integ, err := store.Get("test-integration")
	if err != nil {
		t.Fatalf("failed to get integration: %v", err)
	}

	if integ.Manifest.ID != "test-integration" {
		t.Errorf("expected ID 'test-integration', got %q", integ.Manifest.ID)
	}

	if integ.Manifest.Name != "Test Integration" {
		t.Errorf("expected Name 'Test Integration', got %q", integ.Manifest.Name)
	}

	if integ.Manifest.Category != "observability" {
		t.Errorf("expected Category 'observability', got %q", integ.Manifest.Category)
	}

	if len(integ.Manifest.Symptoms) != 2 {
		t.Errorf("expected 2 symptoms, got %d", len(integ.Manifest.Symptoms))
	}

	// Test sections
	if len(integ.Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(integ.Sections))
	}

	if integ.Sections[0].Title != "Setup Guide" {
		t.Errorf("expected first section title 'Setup Guide', got %q", integ.Sections[0].Title)
	}

	if integ.Sections[0].Order != 1 {
		t.Errorf("expected first section order 1, got %d", integ.Sections[0].Order)
	}

	if integ.Sections[1].Title != "Monitors" {
		t.Errorf("expected second section title 'Monitors', got %q", integ.Sections[1].Title)
	}

	if integ.Sections[1].Order != 2 {
		t.Errorf("expected second section order 2, got %d", integ.Sections[1].Order)
	}

	if len(integ.Sections[0].Content) == 0 {
		t.Errorf("expected non-empty content in first section")
	}

	if len(integ.Sections[1].Content) == 0 {
		t.Errorf("expected non-empty content in second section")
	}

	if integ.Overview == "" {
		t.Errorf("expected non-empty overview content")
	}

	// Test ListCategories
	categories, err := store.ListCategories()
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}

	if len(categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(categories))
	}

	if categories[0].Category != "observability" {
		t.Errorf("expected category 'observability', got %q", categories[0].Category)
	}

	if categories[0].Count != 1 {
		t.Errorf("expected count 1, got %d", categories[0].Count)
	}

	// Test Search
	results, err := store.Search("observability", "test")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test integration directory
	integDir := filepath.Join(tmpDir, "test-int")
	if err := os.Mkdir(integDir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Write minimal manifest
	manifestContent := `id: test-int
name: Test
version: 1.0.0
category: database
description: Test integration
tags: [test]
symptoms: [issue]`

	if err := os.WriteFile(filepath.Join(integDir, "manifest.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("failed to write manifest: %v", err)
	}

	// Write a markdown file with frontmatter
	mdContent := `---
title: "Doc"
order: 0
---
Some content here.`

	if err := os.WriteFile(filepath.Join(integDir, "doc.md"), []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to write doc: %v", err)
	}

	// Create store and service
	store, err := NewStore(tmpDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	svc := NewCatalogService(store)

	// Test ListCategories
	ctx := context.Background()
	categories, err := svc.ListCategories(ctx)
	if err != nil {
		t.Fatalf("failed to list categories: %v", err)
	}

	if len(categories) != 1 {
		t.Errorf("expected 1 category, got %d", len(categories))
	}

	// Test SearchIntegrations
	results, err := svc.SearchIntegrations(ctx, "database", "")
	if err != nil {
		t.Fatalf("failed to search: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Test GetIntegration
	integ, err := svc.GetIntegration(ctx, "test-int")
	if err != nil {
		t.Fatalf("failed to get integration: %v", err)
	}

	if integ.Manifest.ID != "test-int" {
		t.Errorf("expected ID 'test-int', got %q", integ.Manifest.ID)
	}

	if len(integ.Sections) != 1 {
		t.Errorf("expected 1 section, got %d", len(integ.Sections))
	}

	// Test GetIntegration with non-existent ID
	_, err = svc.GetIntegration(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
