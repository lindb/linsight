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

package service

import (
	"context"
	"fmt"

	agentapi "github.com/lindb/linsight/ai/api"
	agentmodel "github.com/lindb/linsight/ai/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

// MemoryService manages Memory CRUD scoped to the caller's organization.
// All operations are strictly org-isolated — no cross-org access.
// The in-memory prompt cache is refreshed by DBMemoryBackend's periodic ticker,
// not by individual writes.
type MemoryService interface {
	// ListMemories returns all memories for the org in ctx.
	ListMemories(ctx context.Context) ([]*agentmodel.Memory, error)
	// GetMemoryByUID returns a single memory owned by the org in ctx.
	GetMemoryByUID(ctx context.Context, uid string) (*agentmodel.Memory, error)
	// CreateMemory inserts a new memory and returns its generated UID.
	CreateMemory(ctx context.Context, req *agentapi.MemoryRequest) (string, error)
	// UpdateMemory applies a full replace update to an existing memory within the caller's org.
	UpdateMemory(ctx context.Context, uid string, req *agentapi.MemoryRequest) error
	// DeleteMemory removes a memory within the caller's org.
	DeleteMemory(ctx context.Context, uid string) error

	// SaveMemoryFromTool is called by the save_memory Eino tool. It creates or updates
	// a single memory entry and returns a human-readable diff describing what changed.
	// id is the memory UID (empty = create new, non-empty = update existing).
	SaveMemoryFromTool(ctx context.Context, id, content, category string) (string, error)
	// DeleteMemoryFromTool is called by the delete_memory Eino tool. It removes the
	// memory identified by uid and returns a human-readable diff describing what was deleted.
	DeleteMemoryFromTool(ctx context.Context, uid string) (string, error)
}

type memoryService struct {
	db dbpkg.DB
}

// NewMemoryService creates a MemoryService backed by db.
func NewMemoryService(db dbpkg.DB) MemoryService {
	return &memoryService{db: db}
}

// ListMemories returns all memories owned by the caller's org.
func (s *memoryService) ListMemories(ctx context.Context) ([]*agentmodel.Memory, error) {
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return nil, ErrNoUser
	}
	orgID := user.Org.ID
	var memories []*agentmodel.Memory
	if err := s.db.RawDB().
		Where("org_id = ?", orgID).
		Order("category asc, id asc").
		Find(&memories).Error; err != nil {
		return nil, err
	}
	return memories, nil
}

// GetMemoryByUID returns a memory owned by the caller's org.
func (s *memoryService) GetMemoryByUID(ctx context.Context, uid string) (*agentmodel.Memory, error) {
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return nil, ErrNoUser
	}
	orgID := user.Org.ID
	var mem agentmodel.Memory
	if err := s.db.Get(&mem, "uid = ? and org_id = ?", uid, orgID); err != nil {
		return nil, err
	}
	return &mem, nil
}

// CreateMemory builds a Memory from req and inserts it.
func (s *memoryService) CreateMemory(ctx context.Context, req *agentapi.MemoryRequest) (string, error) {
	if req.Content == "" {
		return "", fmt.Errorf("content is required")
	}
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return "", ErrNoUser
	}

	mem := &agentmodel.Memory{
		Content:   req.Content,
		Category:  req.Category,
		IsEnabled: req.IsEnabled,
	}
	mem.UID = uuid.GenerateShortUUID()
	mem.OrgID = user.Org.ID
	mem.CreatedBy = user.User.ID
	mem.UpdatedBy = user.User.ID
	if err := s.db.Create(mem); err != nil {
		return "", err
	}
	return mem.UID, nil
}

// UpdateMemory replaces the mutable fields of an existing memory.
func (s *memoryService) UpdateMemory(ctx context.Context, uid string, req *agentapi.MemoryRequest) error {
	if req.Content == "" {
		return fmt.Errorf("content is required")
	}
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return ErrNoUser
	}

	existing, err := s.GetMemoryByUID(ctx, uid)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"content":    req.Content,
		"category":   req.Category,
		"is_enabled": req.IsEnabled,
		"updated_by": user.User.ID,
	}
	if err := s.db.RawDB().Model(&agentmodel.Memory{}).
		Where("uid = ? and org_id = ?", uid, existing.OrgID).
		Updates(updates).Error; err != nil {
		return err
	}
	return nil
}

// DeleteMemory removes a memory owned by the caller's org.
func (s *memoryService) DeleteMemory(ctx context.Context, uid string) error {
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return ErrNoUser
	}
	existing, err := s.GetMemoryByUID(ctx, uid)
	if err != nil {
		return err
	}
	if err := s.db.Delete(&agentmodel.Memory{}, "uid = ? and org_id = ?", uid, existing.OrgID); err != nil {
		return err
	}
	return nil
}

// SaveMemoryFromTool creates or updates a memory entry from the LLM tool call.
// id empty = create new entry; id non-empty = update the existing entry with that UID.
// Returns a diff string describing what was changed, suitable for tool result output.
func (s *memoryService) SaveMemoryFromTool(ctx context.Context, id, content, category string) (string, error) {
	if content == "" {
		return "", fmt.Errorf("content is required")
	}
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return "", ErrNoUser
	}
	orgID := user.Org.ID

	if id == "" {
		// CREATE — no existing entry.
		mem := &agentmodel.Memory{
			Content:   content,
			Category:  category,
			IsEnabled: true,
		}
		mem.UID = uuid.GenerateShortUUID()
		mem.OrgID = orgID
		mem.CreatedBy = user.User.ID
		mem.UpdatedBy = user.User.ID
		if err := s.db.Create(mem); err != nil {
			return "", fmt.Errorf("create memory: %w", err)
		}
		return fmt.Sprintf("Memory created [%s]: %s", mem.UID, content), nil
	}

	// UPDATE — load existing entry first.
	var existing agentmodel.Memory
	if err := s.db.Get(&existing, "uid = ? and org_id = ?", id, orgID); err != nil {
		return "", fmt.Errorf("memory %q not found in your org: %w", id, err)
	}

	oldContent := existing.Content
	oldCategory := existing.Category

	updates := map[string]interface{}{
		"content":    content,
		"category":   category,
		"updated_by": user.User.ID,
	}
	if err := s.db.RawDB().Model(&agentmodel.Memory{}).
		Where("uid = ? and org_id = ?", id, orgID).
		Updates(updates).Error; err != nil {
		return "", fmt.Errorf("update memory: %w", err)
	}

	// Build a human-readable diff.
	var diff string
	if oldCategory != category {
		diff = fmt.Sprintf("Memory updated [%s]\n- Old: %s\n- New: %s\n- Category: %s -> %s",
			id, oldContent, content, oldCategory, category)
	} else {
		diff = fmt.Sprintf("Memory updated [%s]\n- Old: %s\n- New: %s",
			id, oldContent, content)
	}
	return diff, nil
}

// DeleteMemoryFromTool removes a memory entry from the LLM tool call.
// Returns a diff string describing what was deleted.
func (s *memoryService) DeleteMemoryFromTool(ctx context.Context, uid string) (string, error) {
	user := util.GetUser(ctx)
	if user == nil || user.Org == nil {
		return "", ErrNoUser
	}
	orgID := user.Org.ID

	var existing agentmodel.Memory
	if err := s.db.Get(&existing, "uid = ? and org_id = ?", uid, orgID); err != nil {
		return "", fmt.Errorf("memory %q not found in your org: %w", uid, err)
	}

	if err := s.db.Delete(&agentmodel.Memory{}, "uid = ? and org_id = ?", uid, orgID); err != nil {
		return "", fmt.Errorf("delete memory: %w", err)
	}

	return fmt.Sprintf("Memory deleted [%s]: %s", uid, existing.Content), nil
}
