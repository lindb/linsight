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

// GlobalRuleOrgID is the sentinel org_id for a global rule that applies to every
// organisation. Only super admins (RoleLin) may create or modify rules with this
// org_id; per-org rules carry the owning org's real ID.
const GlobalRuleOrgID int64 = 0

// AgentRule is a DB-backed, hot-reloadable global/per-org rule injected into the
// LLM context (similar to Claude Code's .clauderc/rules). Content is a full
// Markdown document whose optional YAML frontmatter carries the dynamic-clipping
// metadata (trigger_tools / trigger_keywords / variables). The API layer stores
// Content verbatim and never parses it — parsing happens once in the RuleEngine
// at cache-load time.
type AgentRule struct {
	basemodel.BaseModel

	// UID is the external identifier used by management APIs.
	UID string `json:"uid" gorm:"column:uid;uniqueIndex"`

	// OrgID scopes the rule. GlobalRuleOrgID (0) means the rule applies to all orgs;
	// a positive value means it is private to that org. Combined with RuleName it
	// forms a unique key so the same rule name can exist globally and per-org.
	OrgID int64 `json:"orgId" gorm:"column:org_id;uniqueIndex:idx_org_rule_name;index"`

	// IsGlobal is a redundant flag mirroring OrgID==GlobalRuleOrgID. It lets the
	// service layer and admin UI filter/validate global rules without special-casing 0.
	IsGlobal bool `json:"isGlobal" gorm:"column:is_global"`

	// RuleName is the human-readable unique name within the (OrgID) scope.
	RuleName string `json:"ruleName" gorm:"column:rule_name;uniqueIndex:idx_org_rule_name"`

	// Content is the full Markdown rule body, including optional YAML frontmatter.
	Content string `json:"content" gorm:"column:content;type:text"`

	// Priority orders rules when multiple rules are matched and concatenated (asc).
	Priority int `json:"priority" gorm:"column:priority"`

	// IsEnabled gates whether the rule is loaded into the runtime cache.
	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled"`
}

// TableName returns the GORM table name.
func (AgentRule) TableName() string { return "agent_rules" }
