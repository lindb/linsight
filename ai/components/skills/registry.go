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

// Package skills implements the three-level skill governance funnel
// (global registry → org subscription → conversation enablement) and
// provides SkillRegistry for loading the effective tool set per conversation.
package skills

import (
	"context"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/lindb/common/pkg/logger"

	agentmodel "github.com/lindb/linsight/ai/model"
	toolscomp "github.com/lindb/linsight/ai/components/tools"
	dbpkg "github.com/lindb/linsight/pkg/db"
)

var log = logger.GetLogger("agent", "skills")

// SkillMeta carries per-tool permission metadata used by the OTel telemetry handler.
type SkillMeta struct {
	// Name is the MCP tool name as registered in agent_skills and presented to the LLM.
	Name string
	// ControlLevel indicates the governance tier that authorized this tool for the request.
	ControlLevel agentmodel.SkillControlLevel
}

// skillRow is a raw result row from the three-table funnel LEFT JOIN query.
type skillRow struct {
	Name       string
	IsGlobal   bool
	OrgStatus  int
	ConvStatus int
}

// SkillRegistry loads the effective tool set for a conversation by evaluating
// the three-level governance funnel:
//
//	global registry (agent_skills)
//	  → org subscription (agent_org_skills)
//	    → conversation enablement (agent_conversation_skills)
//
// Tool names that pass the funnel are matched against the live MCP tool pool.
// Non-matching tools are hard-blocked: they are never sent to the LLM.
type SkillRegistry struct {
	db           dbpkg.DB
	toolsFactory *toolscomp.MCPToolsFactory
}

// NewSkillRegistry creates a SkillRegistry backed by db and the org's MCP tool pool.
func NewSkillRegistry(db dbpkg.DB, toolsFactory *toolscomp.MCPToolsFactory) *SkillRegistry {
	return &SkillRegistry{db: db, toolsFactory: toolsFactory}
}

// LoadConversationTools returns the tool slice and per-tool metadata that the agent may
// use for the given org + conversation. Governance rules:
//
//   - is_global=1  → auto-included regardless of subscription; ControlLevel="global"
//   - is_global=0  → requires org_skill.status=1 AND conv_skill.status=1; ControlLevel="conversation"
//   - MCP tools not registered in agent_skills are excluded (hard-blocked)
//
// If the agent_skills table is empty (skill governance not yet configured),
// all org MCP tools are returned unfiltered with ControlLevel="global" as a safe default.
//
// Failures from the MCP layer are fail-open: a warning is logged and an empty tool list
// is returned so the agent can still respond without tools.
func (r *SkillRegistry) LoadConversationTools(
	ctx context.Context, orgID, conversationID int64,
) ([]tool.BaseTool, []SkillMeta, error) {
	// Quick existence check: if no skills have been registered, operate in uncontrolled mode.
	var checkSkills []*agentmodel.Skill
	if err := r.db.RawDB().Limit(1).Find(&checkSkills).Error; err != nil {
		log.Warn("skill registry: failed to check agent_skills table, falling back to all tools", logger.Error(err))
		return r.fallbackToAllTools(ctx, orgID)
	}
	if len(checkSkills) == 0 {
		// Governance not configured — return all org tools so the agent is functional by default.
		return r.fallbackToAllTools(ctx, orgID)
	}

	// Three-table funnel: resolve which skill names are allowed for this org+conversation.
	//
	// A skill passes the funnel when:
	//   (a) is_global = 1 (no subscription needed), OR
	//   (b) org has an active subscription (org_status=1) AND the conversation explicitly enabled it (conv_status=1)
	const query = `
SELECT
    s.name,
    s.is_global,
    COALESCE(os.status, 0) AS org_status,
    COALESCE(cs.status, 0) AS conv_status
FROM agent_skills AS s
LEFT JOIN agent_org_skills AS os
    ON os.skill_id = s.id AND os.org_id = ?
LEFT JOIN agent_conversation_skills AS cs
    ON cs.skill_id = s.id AND cs.conversation_id = ?
WHERE s.is_global = 1
   OR (COALESCE(os.status, 0) = 1 AND COALESCE(cs.status, 0) = 1)`

	var rows []skillRow
	if err := r.db.RawDB().Raw(query, orgID, conversationID).Scan(&rows).Error; err != nil {
		log.Warn("skill registry: funnel query failed, falling back to all tools", logger.Error(err))
		return r.fallbackToAllTools(ctx, orgID)
	}

	if len(rows) == 0 {
		// Skills table has entries but none passed the funnel — strict block.
		log.Debug("skill registry: no skills passed the governance funnel",
			logger.Int64("org_id", orgID),
			logger.Int64("conversation_id", conversationID),
		)
		return nil, nil, nil
	}

	// Build a name → ControlLevel map for fast lookup during MCP tool filtering.
	allowed := make(map[string]agentmodel.SkillControlLevel, len(rows))
	for _, r := range rows {
		if r.IsGlobal {
			allowed[r.Name] = agentmodel.SkillControlLevelGlobal
		} else {
			allowed[r.Name] = agentmodel.SkillControlLevelConversation
		}
	}

	// Fetch all MCP tools available to the org, then filter by allowed names.
	orgTools, err := r.toolsFactory.GetToolsForOrg(ctx, orgID)
	if err != nil {
		log.Warn("skill registry: failed to load org MCP tools", logger.Error(err))
		return nil, nil, nil
	}

	return r.filterByAllowed(ctx, orgTools, allowed)
}

// filterByAllowed filters orgTools to only those whose names appear in the allowed map,
// and returns the matched tools alongside their SkillMeta.
func (r *SkillRegistry) filterByAllowed(
	ctx context.Context,
	orgTools []tool.BaseTool,
	allowed map[string]agentmodel.SkillControlLevel,
) ([]tool.BaseTool, []SkillMeta, error) {
	// Use a short timeout for Info() calls — these are expected to be in-memory for MCP tools.
	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filtered := make([]tool.BaseTool, 0, len(allowed))
	meta := make([]SkillMeta, 0, len(allowed))

	for _, t := range orgTools {
		info, err := t.Info(infoCtx)
		if err != nil {
			log.Warn("skill registry: failed to get tool info, skipping", logger.Error(err))
			continue
		}
		level, ok := allowed[info.Name]
		if !ok {
			// Tool is not in the allowed set — hard-block it.
			continue
		}
		filtered = append(filtered, t)
		meta = append(meta, SkillMeta{Name: info.Name, ControlLevel: level})
	}
	return filtered, meta, nil
}

// fallbackToAllTools returns all org MCP tools unfiltered, each tagged as globally controlled.
// Called when skill governance is not yet configured (empty agent_skills table) or when
// an unexpected DB error prevents the funnel query from running.
func (r *SkillRegistry) fallbackToAllTools(ctx context.Context, orgID int64) ([]tool.BaseTool, []SkillMeta, error) {
	orgTools, err := r.toolsFactory.GetToolsForOrg(ctx, orgID)
	if err != nil {
		log.Warn("skill registry: fallback tool load failed", logger.Error(err))
		return nil, nil, nil
	}
	if len(orgTools) == 0 {
		return nil, nil, nil
	}

	infoCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	meta := make([]SkillMeta, 0, len(orgTools))
	for _, t := range orgTools {
		info, err := t.Info(infoCtx)
		if err != nil {
			log.Warn("skill registry: failed to get tool info in fallback, skipping", logger.Error(err))
			continue
		}
		meta = append(meta, SkillMeta{Name: info.Name, ControlLevel: agentmodel.SkillControlLevelGlobal})
	}
	return orgTools, meta, nil
}
