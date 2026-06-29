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

// SkillControlLevel indicates the governance tier that authorized a skill for the current request.
type SkillControlLevel string

const (
	// SkillControlLevelGlobal means the skill is injected unconditionally for every request.
	// No org subscription or per-conversation enablement is required.
	SkillControlLevelGlobal SkillControlLevel = "global"

	// SkillControlLevelConversation means the skill was explicitly enabled for the conversation.
	// Both an org subscription (agent_org_skills.status=1) and a per-conversation flag
	// (agent_conversation_skills.status=1) are required.
	SkillControlLevelConversation SkillControlLevel = "conversation"
)

// Skill is the system-wide registry entry for a named skill.
// Its Name must match the MCP tool name used by the LLM so that the three-level
// funnel can filter the live tool list returned by MCPToolsFactory.
type Skill struct {
	basemodel.BaseModel

	// Name is the canonical tool name exposed to the LLM; must match the MCP tool name.
	Name string `json:"name" gorm:"column:name;uniqueIndex"`

	Description string `json:"description" gorm:"column:description;type:text"`

	// IsGlobal true means the skill is included unconditionally without any subscription.
	// Set true for low-risk read-only skills (e.g., metric queries).
	IsGlobal bool `json:"isGlobal" gorm:"column:is_global"`

	// SkillType is a free-form tag (e.g., "mcp", "builtin") for UI grouping.
	SkillType string `json:"skillType" gorm:"column:skill_type"`
}

// TableName returns the GORM table name.
func (Skill) TableName() string { return "skills" }

// OrgSkill records whether an org has subscribed to a non-global skill.
// Status=1 means active; any other value means inactive.
type OrgSkill struct {
	basemodel.BaseModel

	OrgID   int64 `json:"orgId" gorm:"column:org_id;index"`
	SkillID int64 `json:"skillId" gorm:"column:skill_id;index"`
	Status  int   `json:"status" gorm:"column:status"`
}

// TableName returns the GORM table name.
func (OrgSkill) TableName() string { return "org_skills" }

// ConversationSkill records whether a non-global skill has been explicitly enabled
// for a specific conversation by the user. Both the org subscription and this record
// must be active (status=1) for the skill to pass the conversation-level gate.
type ConversationSkill struct {
	basemodel.BaseModel

	ConversationID int64 `json:"conversationId" gorm:"column:conversation_id;index"`
	SkillID        int64 `json:"skillId" gorm:"column:skill_id;index"`
	Status         int   `json:"status" gorm:"column:status"`
}

// TableName returns the GORM table name.
func (ConversationSkill) TableName() string { return "conversation_skills" }
