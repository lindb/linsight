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

package model

import (
	basemodel "github.com/lindb/linsight/model"
)

// Memory represents a single atomic long-term memory entry scoped to an organization.
// Each row is one independent fact; all enabled entries for an org are aggregated into
// a <user_memory> block and injected into the system prompt at conversation start.
// The LLM references individual entries by their short UID when issuing save/delete
// tool calls.
type Memory struct {
	basemodel.BaseModel

	// UID is the external short identifier used by management APIs and tool calls.
	UID string `json:"uid" gorm:"column:uid;uniqueIndex"`
	// OrgID scopes the memory to a single organisation. Memories are never shared across orgs.
	OrgID int64 `json:"orgId" gorm:"column:org_id;index"`
	// Content is the memory text. This is what gets rendered into the system prompt.
	Content string `json:"content" gorm:"column:content;type:text"`
	// Category is an optional tag for grouping related memories (e.g. "preference", "project").
	Category string `json:"category,omitempty" gorm:"column:category;index"`
	// IsEnabled gates whether the memory is loaded into the prompt cache.
	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled"`
}

// TableName returns the GORM table name.
func (Memory) TableName() string { return "memories" }
