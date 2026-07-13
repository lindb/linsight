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

// SkillContextMode defines how the host agent should execute a skill when invoked.
// Aligned with github.com/cloudwego/eino/adk/middlewares/skill.ContextMode.
type SkillContextMode string

const (
	// SkillContextModeInline — the skill content is returned directly as a tool result;
	// the host agent absorbs it and writes the final answer in the same turn.
	SkillContextModeInline SkillContextMode = ""
	// SkillContextModeFork — creates an isolated sub-agent without parent conversation
	// history; receives only the skill's instructions as input.
	SkillContextModeFork SkillContextMode = "fork"
	// SkillContextModeForkWithContext — creates a sub-agent that inherits the parent
	// conversation history for deep, context-aware reasoning.
	SkillContextModeForkWithContext SkillContextMode = "fork_with_context"
)

// Skill is the system-wide registry entry for a named skill. Metadata
// (description, context_mode, agent) is stored in the Content's YAML frontmatter
// and parsed by DBBackend at cache-load time.
type Skill struct {
	basemodel.BaseModel

	// UID is the external identifier used by management APIs.
	UID string `json:"uid" gorm:"column:uid;uniqueIndex"`
	// OrgID scopes the skill. GlobalRuleOrgID (0) means the skill applies to all orgs.
	OrgID int64 `json:"orgId" gorm:"column:org_id;index"`
	// Name is the canonical skill name used as the tool parameter value.
	Name string `json:"name" gorm:"column:name;uniqueIndex"`
	// Content is the full Markdown document. The optional YAML frontmatter carries
	// description, context_mode, and agent metadata; the body is the skill body.
	Content string `json:"content" gorm:"column:content;type:text"`
	// IsGlobal true means the skill is available to all orgs without subscription.
	IsGlobal bool `json:"isGlobal" gorm:"column:is_global"`
	// IsEnabled gates whether the skill is loaded by DBBackend.
	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled"`
}

// TableName returns the GORM table name.
func (Skill) TableName() string { return "skills" }
