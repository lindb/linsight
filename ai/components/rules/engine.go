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

// Package rules implements a DB-backed, hot-reloadable global/per-org rule
// injection mechanism for the AI agent (similar to Claude Code's .clauderc/rules).
//
// Rules are stored in the agent_rules table as full Markdown documents. An optional
// YAML frontmatter block carries the dynamic-clipping metadata:
//
//	---
//	trigger_tools: [linsight_datasource_query]
//	trigger_keywords: [sql, query]
//	variables: {"env": "prod"}
//	---
//	### Constraints
//	Always confirm the target environment before running any query...
//
// The engine loads rules into an in-memory cache (RWMutex-guarded) and, on each LLM
// call, matches them against the tools the session has actually invoked and the
// conversation text, then concatenates the matched rules into a single Markdown
// string. Variables are emitted verbatim as JSON — the engine never templates them;
// the LLM aligns them via in-context learning.
package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/lindb/common/pkg/logger"
	"gopkg.in/yaml.v3"

	agentmodel "github.com/lindb/linsight/ai/model"
	aiutil "github.com/lindb/linsight/ai/util"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
)

var ruleLog = logger.GetLogger("agent", "rules")

// RuleEngine loads rules from a backing store into memory and compiles the subset
// matched by the current session into an injectable Markdown string. DB is one
// implementation (DBRuleEngine); the interface leaves room for others.
type RuleEngine interface {
	// RefreshCache reloads all enabled rules from the store into the in-memory cache.
	// It is safe to call concurrently with MatchAndCompile.
	RefreshCache(ctx context.Context) error
	// MatchAndCompile returns the Markdown text of all rules matched by the current
	// session (empty string if none). The org is extracted from ctx; the invoked
	// tools and conversation text are extracted from input.
	MatchAndCompile(ctx context.Context, input *einomodel.CallbackInput) (string, error)
}

// RuntimeRule is the pre-parsed, cache-resident form of an AgentRule. All frontmatter
// parsing happens once at load time so the request hot-path performs no parsing.
type RuntimeRule struct {
	RuleName string
	Priority int
	// Body is the Markdown content with the frontmatter block stripped.
	Body string
	// VariablesJSON is the frontmatter "variables" re-serialised to a compact JSON
	// string, emitted verbatim to the LLM. Empty when no variables were declared.
	VariablesJSON string
	// TriggerTools / TriggerKeywords are lower-cased for case-insensitive matching.
	// A rule with both empty is an always-on (resident) rule.
	TriggerTools    []string
	TriggerKeywords []string
}

// ruleFrontmatter mirrors the optional YAML frontmatter of a rule's Content.
type ruleFrontmatter struct {
	TriggerTools    []string `yaml:"trigger_tools"`
	TriggerKeywords []string `yaml:"trigger_keywords"`
	// Variables is decoded generically so any JSON/YAML shape is accepted and
	// re-emitted as JSON without the engine imposing a schema.
	Variables any `yaml:"variables"`
}

// DBRuleEngine is a RuleEngine backed by the agent_rules table with an in-memory,
// RWMutex-guarded cache. Global rules (org_id=0) apply to every org; per-org rules
// are bucketed by org_id.
type DBRuleEngine struct {
	db              dbpkg.DB
	refreshInterval time.Duration

	mu          sync.RWMutex
	globalRules []*RuntimeRule
	orgRules    map[int64][]*RuntimeRule
}

// NewDBRuleEngine creates a DBRuleEngine. Call RefreshCache once before serving and
// optionally StartAutoRefresh for periodic reloads.
func NewDBRuleEngine(db dbpkg.DB, refreshInterval time.Duration) *DBRuleEngine {
	return &DBRuleEngine{
		db:              db,
		refreshInterval: refreshInterval,
		orgRules:        make(map[int64][]*RuntimeRule),
	}
}

// StartAutoRefresh launches a background goroutine that reloads the cache every
// refreshInterval until ctx is cancelled. A non-positive interval disables the ticker.
func (e *DBRuleEngine) StartAutoRefresh(ctx context.Context) {
	if e.refreshInterval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(e.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := e.RefreshCache(ctx); err != nil {
					ruleLog.Warn("rules: periodic cache refresh failed", logger.Error(err))
				}
			}
		}
	}()
}

// RefreshCache reloads all enabled rules and atomically swaps the in-memory cache.
// A malformed rule is skipped (with a warning) rather than failing the whole reload.
func (e *DBRuleEngine) RefreshCache(ctx context.Context) error {
	var rows []*agentmodel.AgentRule
	if err := e.db.RawDB().WithContext(ctx).
		Where("is_enabled = ?", true).
		Order("priority asc, id asc").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("load agent_rules: %w", err)
	}

	global := make([]*RuntimeRule, 0)
	perOrg := make(map[int64][]*RuntimeRule)
	for _, row := range rows {
		rr, err := parseRuleContent(row.Content)
		if err != nil {
			ruleLog.Warn("rules: skipping rule with malformed frontmatter",
				logger.String("rule", row.RuleName), logger.Error(err))
			continue
		}
		rr.RuleName = row.RuleName
		rr.Priority = row.Priority
		if row.OrgID == agentmodel.GlobalRuleOrgID {
			global = append(global, rr)
		} else {
			perOrg[row.OrgID] = append(perOrg[row.OrgID], rr)
		}
	}

	e.mu.Lock()
	e.globalRules = global
	e.orgRules = perOrg
	e.mu.Unlock()

	ruleLog.Info("rules: cache refreshed",
		logger.Int("global", len(global)), logger.Int("orgs", len(perOrg)))
	return nil
}

// MatchAndCompile selects the rules matched by the current session and concatenates
// them into a single Markdown block. Returns "" when nothing matches.
func (e *DBRuleEngine) MatchAndCompile(ctx context.Context, input *einomodel.CallbackInput) (string, error) {
	if input == nil {
		return "", nil
	}
	orgID := int64(0)
	if u := util.GetUser(ctx); u != nil && u.Org != nil {
		orgID = u.Org.ID
	}

	// Snapshot candidate rules under the read lock, then release it before matching.
	e.mu.RLock()
	candidates := make([]*RuntimeRule, 0, len(e.globalRules)+len(e.orgRules[orgID]))
	candidates = append(candidates, e.globalRules...)
	candidates = append(candidates, e.orgRules[orgID]...)
	e.mu.RUnlock()

	if len(candidates) == 0 {
		return "", nil
	}

	invokedTools, contextText := extractSessionSignals(input.Messages)

	matched := make([]*RuntimeRule, 0, len(candidates))
	for _, r := range candidates {
		if r.matches(invokedTools, contextText) {
			matched = append(matched, r)
		}
	}
	if len(matched) == 0 {
		return "", nil
	}

	// Stable order: priority asc, then name, so identical inputs yield identical prompts.
	sort.SliceStable(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority < matched[j].Priority
		}
		return matched[i].RuleName < matched[j].RuleName
	})

	return compileRules(matched), nil
}

// matches implements the hit logic: a rule with no triggers is always-on; otherwise
// it matches if any declared tool was invoked or any declared keyword appears in text.
func (r *RuntimeRule) matches(invokedTools map[string]struct{}, contextText string) bool {
	if len(r.TriggerTools) == 0 && len(r.TriggerKeywords) == 0 {
		return true
	}
	// Tool matching only when trigger_tools is declared.
	for _, t := range r.TriggerTools {
		if _, ok := invokedTools[t]; ok {
			return true
		}
	}
	for _, kw := range r.TriggerKeywords {
		if kw != "" && strings.Contains(contextText, kw) {
			return true
		}
	}
	return false
}

// extractSessionSignals returns the set of tool names actually invoked in the session
// (from assistant ToolCalls, lower-cased) and the lower-cased concatenation of all
// message text (used for keyword matching).
func extractSessionSignals(messages []*schema.Message) (map[string]struct{}, string) {
	invoked := make(map[string]struct{})
	var textSB strings.Builder
	for _, m := range messages {
		if m == nil {
			continue
		}
		if m.Content != "" {
			textSB.WriteString(m.Content)
			textSB.WriteString("\n")
		}
		for _, tc := range m.ToolCalls {
			if name := strings.ToLower(strings.TrimSpace(tc.Function.Name)); name != "" {
				invoked[name] = struct{}{}
			}
		}
	}
	return invoked, strings.ToLower(textSB.String())
}

// compileRules renders matched rules into the agreed Markdown format. Variables are
// emitted verbatim (as JSON) — never parsed or templated.
func compileRules(rules []*RuntimeRule) string {
	var sb strings.Builder
	for _, r := range rules {
		sb.WriteString("## Rule: ")
		sb.WriteString(r.RuleName)
		sb.WriteString("\n")
		if r.VariablesJSON != "" {
			sb.WriteString("- Environment variables: ")
			sb.WriteString(r.VariablesJSON)
			sb.WriteString("\n")
		}
		sb.WriteString("\n### Constraints:\n")
		sb.WriteString(strings.TrimSpace(r.Body))
		sb.WriteString("\n---\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ValidateContent checks that a rule's Content has well-formed (or absent) YAML
// frontmatter. Management services call it to reject malformed rules at write time
// rather than silently dropping them at cache-load time.
func ValidateContent(content string) error {
	_, err := parseRuleContent(content)
	return err
}

// parseRuleContent splits an optional leading YAML frontmatter block from the Markdown
// body and builds a RuntimeRule. When no frontmatter is present the whole content is
// treated as the body (an always-on rule).
func parseRuleContent(content string) (*RuntimeRule, error) {
	fm, body := aiutil.SplitFrontmatter(content)
	rr := &RuntimeRule{Body: body}
	if fm != "" {
		var meta ruleFrontmatter
		if uerr := yaml.Unmarshal([]byte(fm), &meta); uerr != nil {
			return nil, fmt.Errorf("parse frontmatter: %w", uerr)
		}
		rr.TriggerTools = lowerAll(meta.TriggerTools)
		rr.TriggerKeywords = lowerAll(meta.TriggerKeywords)
		if meta.Variables != nil {
			// Re-emit variables as compact JSON so the LLM receives a stable,
			// language-neutral representation regardless of YAML vs. inline-JSON input.
			if raw, jerr := json.Marshal(normalizeYAML(meta.Variables)); jerr == nil {
				rr.VariablesJSON = string(raw)
			} else {
				ruleLog.Warn("rules: failed to serialise variables, dropping", logger.Error(jerr))
			}
		}
	}
	return rr, nil
}

// lowerAll returns a new slice with each element trimmed and lower-cased; empty
// elements are dropped.
func lowerAll(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.ToLower(strings.TrimSpace(s))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// normalizeYAML converts yaml.v3's map[interface{}]interface{} nodes into
// map[string]interface{} recursively so encoding/json can marshal them.
func normalizeYAML(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = normalizeYAML(val)
		}
		return m
	case map[any]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[fmt.Sprintf("%v", k)] = normalizeYAML(val)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, val := range t {
			s[i] = normalizeYAML(val)
		}
		return s
	default:
		return v
	}
}
