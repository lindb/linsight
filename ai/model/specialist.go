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
	"fmt"
	"regexp"

	basemodel "github.com/lindb/linsight/model"
)

// specialistNamePattern constrains a Specialist Name to a tool-name-safe slug: it
// becomes the suffix of the handoff tool name (handoff_to_<name>), which the LLM
// emits as a function call, so it must match the OpenAI function-name grammar
// (letters, digits, underscore, dash; must start with a letter).
var specialistNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// maxSpecialistNameLen bounds Name so that "handoff_to_" + Name stays within the OpenAI
// 64-character function-name limit (len("handoff_to_") == 11).
const maxSpecialistNameLen = 64 - len("handoff_to_")

// Specialist is a DB-persisted sub-agent (the Host-Specialist "Specialist" role),
// stored as a Claude-Code-style agent.md document. Each enabled Specialist is exposed to
// the Host LLM as a handoff_to_<name> tool; when invoked, the Specialist runs in an
// isolated context (its system prompt plus the Host-supplied goal only) and returns a
// structured evidence object. Metadata (description, model, tools) lives in the Content's
// YAML frontmatter and is parsed by multiagent.DBBackend at cache-load time; the frontmatter
// body is the specialist's system prompt.
type Specialist struct {
	basemodel.BaseModel

	// UID is the external identifier used by management APIs.
	UID string `json:"uid" gorm:"column:uid;uniqueIndex"`
	// OrgID scopes the specialist. GlobalRuleOrgID (0) means it applies to all orgs.
	OrgID int64 `json:"orgId" gorm:"column:org_id;index"`
	// Name is the canonical specialist name and the suffix of its handoff tool
	// (handoff_to_<name>). Must be a tool-name-safe slug (see specialistNamePattern).
	Name string `json:"name" gorm:"column:name;uniqueIndex"`
	// Content is the full agent.md Markdown document. The optional YAML frontmatter carries
	// description / model / tools; the body is the specialist's system prompt. Parsed by
	// DBBackend at cache-load time; stored verbatim here.
	Content string `json:"content" gorm:"column:content;type:text"`
	// IsGlobal true means the specialist is available to all orgs.
	IsGlobal bool `json:"isGlobal" gorm:"column:is_global"`
	// IsEnabled gates whether the specialist is loaded (and its handoff tool bound).
	IsEnabled bool `json:"isEnabled" gorm:"column:is_enabled"`
}

// TableName returns the GORM table name.
func (Specialist) TableName() string { return "specialists" }

// ValidateName reports whether name is a valid tool-name-safe specialist slug.
// Returns a descriptive error when invalid so the service layer can surface it.
func ValidateName(name string) error {
	if !specialistNamePattern.MatchString(name) {
		return fmt.Errorf("specialist name %q is invalid: must match %s", name, specialistNamePattern.String())
	}
	if len(name) > maxSpecialistNameLen {
		return fmt.Errorf("specialist name %q is too long: max %d characters", name, maxSpecialistNameLen)
	}
	return nil
}
