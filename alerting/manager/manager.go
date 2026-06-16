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

package manager

import (
	"context"
	"encoding/json"
	"regexp"
	"sync"
	"time"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/alerting/template"
	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/service"
)

// AlertManager receives alert events from rule workers, applies silence/inhibit filtering,
// routes them to receivers and manages group-based notification scheduling.
type AlertManager interface {
	// ProcessEvents receives new or changed AlertEvents from rule workers.
	ProcessEvents(orgID int64, events []*model.AlertEvent)
	// ReloadConfig reloads routes, silences and inhibit rules from the database.
	ReloadConfig(orgID int64)
	// StartRepeatTicker runs a background loop that periodically calls MaybeRepeat on all
	// alert groups so that steady-state Firing alerts get re-notified after RepeatInterval.
	StartRepeatTicker(ctx context.Context)
}

// Deps holds all service dependencies needed by the AlertManager.
type Deps struct {
	AlertEventSrv           service.AlertEventService
	AlertRouteSrv           service.AlertRouteService
	ContactPointSrv         service.ContactPointService
	AlertSilenceSrv         service.AlertSilenceService
	AlertInhibitRuleSrv     service.AlertInhibitRuleService
	NotificationTemplateSrv service.NotificationTemplateService
	NotificationSrv         service.NotificationService
	ContactPointStatusSrv   service.ContactPointStatusService // records per-channel delivery outcomes
	RenderingEngine         *template.RenderingEngine        // shared engine with TTL cache for org templates
	ExternalURL             string
}

type alertManager struct {
	deps    *Deps
	groups  map[string]*alertGroup // key: orgID:contactPointUID:groupKey
	groupMu sync.Mutex

	// cached config per org — orgRoutes stores the full routing tree (nil if not yet loaded)
	orgRoutes   map[int64]*model.AlertRouteNode
	orgSilences map[int64][]*model.AlertSilence
	orgInhibits map[int64][]*model.AlertInhibitRule
	cfgMu       sync.RWMutex

	logger logger.Logger
}

// NewAlertManager creates an AlertManager instance.
func NewAlertManager(deps *Deps) AlertManager {
	return &alertManager{
		deps:        deps,
		groups:      make(map[string]*alertGroup),
		orgRoutes:   make(map[int64]*model.AlertRouteNode),
		orgSilences: make(map[int64][]*model.AlertSilence),
		orgInhibits: make(map[int64][]*model.AlertInhibitRule),
		logger:      logger.GetLogger("Alerting", "AlertManager"),
	}
}

func (m *alertManager) ProcessEvents(orgID int64, events []*model.AlertEvent) {
	now := time.Now().UnixMilli()
	m.logger.Info("processing alert events",
		logger.Int64("org", orgID),
		logger.Int32("count", int32(len(events))))

	// persist / update alert events in the database
	for _, ev := range events {
		if err := m.deps.AlertEventSrv.UpsertAlertEvent(orgID, ev); err != nil {
			m.logger.Error("upsert alert event failure", logger.Int64("org", orgID), logger.Error(err))
		}
	}

	silences, inhibits, routeTree := m.getOrgConfig(orgID)

	for _, ev := range events {
		// check silence
		if m.isSilenced(ev, silences, now) {
			continue
		}

		// check inhibition
		if m.isInhibited(ev, inhibits, events) {
			continue
		}

		// route to receiver(s)
		matched := m.traverseRouteTree(ev, routeTree)
		if len(matched) == 0 {
			m.logger.Info("no route matched",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID))
		}
		for _, route := range matched {
			m.logger.Info("alert event routed",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID),
				logger.String("routeUID", route.UID),
				logger.String("contactPointUID", route.ContactPointUID))
			groupKey := m.buildGroupKey(orgID, route, ev)
			m.groupMu.Lock()
			grp, ok := m.groups[groupKey]
			if !ok {
				grp = newAlertGroup(orgID, groupKey, route, m.deps)
				m.groups[groupKey] = grp
			}
			m.groupMu.Unlock()

			grp.addEvent(ev, now)
		}
	}
}

func (m *alertManager) ReloadConfig(orgID int64) {
	now := time.Now().UnixMilli()
	routeTree, err := m.deps.AlertRouteSrv.GetAlertRouteTreeByOrg(orgID)
	if err != nil {
		m.logger.Error("reload routes failure", logger.Int64("org", orgID), logger.Error(err))
		routeTree = nil
	}

	silences, err := m.deps.AlertSilenceSrv.GetActiveSilencesByOrg(orgID, now)
	if err != nil {
		m.logger.Error("reload silences failure", logger.Int64("org", orgID), logger.Error(err))
		silences = nil
	}

	inhibits, err := m.deps.AlertInhibitRuleSrv.GetEnabledInhibitRulesByOrg(orgID)
	if err != nil {
		m.logger.Error("reload inhibit rules failure", logger.Int64("org", orgID), logger.Error(err))
		inhibits = nil
	}

	m.cfgMu.Lock()
	m.orgRoutes[orgID] = routeTree
	m.orgSilences[orgID] = silences
	m.orgInhibits[orgID] = inhibits
	m.cfgMu.Unlock()
}

// getOrgConfig returns a snapshot of the current configuration for an org.
func (m *alertManager) getOrgConfig(orgID int64) ([]*model.AlertSilence, []*model.AlertInhibitRule, *model.AlertRouteNode) {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.orgSilences[orgID], m.orgInhibits[orgID], m.orgRoutes[orgID]
}

// isSilenced returns true if the alert event matches any active silence.
func (m *alertManager) isSilenced(ev *model.AlertEvent, silences []*model.AlertSilence, now int64) bool {
	labels := parseEventLabels(ev)
	for _, s := range silences {
		if s.StartsAt > now || s.EndsAt <= now {
			continue
		}
		matchers, err := s.ToMatchers()
		if err != nil {
			continue
		}
		if matchAll(labels, matchers) {
			m.logger.Info("alert event silenced",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID),
				logger.String("silenceUID", s.UID))
			return true
		}
	}
	return false
}

// isInhibited returns true if any enabled inhibit rule suppresses this event.
func (m *alertManager) isInhibited(ev *model.AlertEvent, inhibits []*model.AlertInhibitRule, allEvents []*model.AlertEvent) bool {
	targetLabels := parseEventLabels(ev)

	for _, rule := range inhibits {
		targetMatchers, err := rule.ToTargetMatchers()
		if err != nil {
			continue
		}
		// check if this event is a target
		if !matchAll(targetLabels, targetMatchers) {
			continue
		}

		equalLabels, err := rule.ToEqualLabels()
		if err != nil {
			continue
		}
		sourceMatchers, err := rule.ToSourceMatchers()
		if err != nil {
			continue
		}

		// look for a firing source alert that satisfies the source matchers and equal-labels
		for _, src := range allEvents {
			if src.Fingerprint == ev.Fingerprint {
				continue
			}
			if src.State != model.AlertEventFiring {
				continue
			}
			srcLabels := parseEventLabels(src)
			if !matchAll(srcLabels, sourceMatchers) {
				continue
			}
			// check equal labels
			inhibit := true
			for _, lbl := range equalLabels {
				if srcLabels[lbl] != targetLabels[lbl] {
					inhibit = false
					break
				}
			}
			if inhibit {
				m.logger.Info("alert event inhibited",
					logger.String("fingerprint", ev.Fingerprint),
					logger.String("rule", ev.RuleUID),
					logger.String("inhibitRuleUID", rule.UID))
				return true
			}
		}
	}
	return false
}

// traverseRouteTree applies the Alertmanager-style tree-traversal algorithm to find all
// routes that should receive notifications for the given event.
//
// Traversal rules (mirrors Grafana / Alertmanager behaviour):
//   - The root node (IsDefault=true) is entered unconditionally.
//   - Child nodes are checked against the event's label matchers in sort_order.
//   - If continue=false on a matched child, sibling evaluation stops after that child.
//   - When no child matches, the current node itself becomes the fallback receiver.
//   - Zero-value GroupWait/GroupInterval/RepeatInterval/GroupBy on a leaf are filled from ancestors.
func (m *alertManager) traverseRouteTree(ev *model.AlertEvent, root *model.AlertRouteNode) []*model.AlertRoute {
	if root == nil {
		return nil
	}
	labels := parseEventLabels(ev)
	return traverseNode(root, labels, nil)
}

// traverseNode is the recursive core of traverseRouteTree.
// ancestors holds the route chain from root to the current node's parent (used for inheritance).
func traverseNode(node *model.AlertRouteNode, labels map[string]string, ancestors []*model.AlertRoute) []*model.AlertRoute {
	// Non-root nodes must match the event labels; root (IsDefault) always passes.
	if !node.IsDefault {
		matchers, err := node.ToMatchers()
		if err != nil || !matchAll(labels, matchers) {
			return nil
		}
	}

	newAncestors := append(ancestors, &node.AlertRoute) //nolint:gocritic // intentional accumulation per call stack frame

	// Evaluate children in sort order
	var result []*model.AlertRoute
	childMatched := false
	for _, child := range node.Children {
		childResults := traverseNode(child, labels, newAncestors)
		if len(childResults) > 0 {
			result = append(result, childResults...)
			childMatched = true
			if !child.Continue {
				// Stop checking siblings once the first non-continue match is found
				break
			}
		}
	}

	// When no child matched, the current node is the fallback/leaf receiver.
	// Inherit zero-value timing fields from ancestors so every dispatched route
	// has meaningful GroupWait / GroupInterval / RepeatInterval / GroupBy values.
	if !childMatched {
		resolved := resolveInheritance(&node.AlertRoute, ancestors)
		result = append(result, resolved)
	}
	return result
}

// resolveInheritance returns a copy of route with zero-value fields filled from the nearest
// ancestor that defines them (walking from closest parent toward root).
func resolveInheritance(route *model.AlertRoute, ancestors []*model.AlertRoute) *model.AlertRoute {
	resolved := *route
	for i := len(ancestors) - 1; i >= 0; i-- {
		p := ancestors[i]
		if resolved.GroupWait == 0 {
			resolved.GroupWait = p.GroupWait
		}
		if resolved.GroupInterval == 0 {
			resolved.GroupInterval = p.GroupInterval
		}
		if resolved.RepeatInterval == 0 {
			resolved.RepeatInterval = p.RepeatInterval
		}
		if len(resolved.GroupBy) == 0 {
			resolved.GroupBy = p.GroupBy
		}
	}
	return &resolved
}

// buildGroupKey constructs the unique key for an alert group.
func (m *alertManager) buildGroupKey(orgID int64, route *model.AlertRoute, ev *model.AlertEvent) string {
	labels := parseEventLabels(ev)
	groupBy, _ := route.ToGroupBy()
	groupValues := make([]byte, 0, 64)
	for _, k := range groupBy {
		groupValues = append(groupValues, k+"="+labels[k]+";"...)
	}
	return string(groupValues) + ":" + route.UID
}

// matchAll returns true when every matcher in the list matches the label set.
func matchAll(labels map[string]string, matchers []model.LabelMatcher) bool {
	for _, m := range matchers {
		val := labels[m.Name]
		switch m.Type {
		case model.MatchEqual:
			if val != m.Value {
				return false
			}
		case model.MatchNotEqual:
			if val == m.Value {
				return false
			}
		case model.MatchRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if !ok {
				return false
			}
		case model.MatchNotRegexp:
			ok, _ := regexp.MatchString("^(?:"+m.Value+")$", val)
			if ok {
				return false
			}
		}
	}
	return true
}

// parseEventLabels decodes the JSON labels of an AlertEvent into a map.
func parseEventLabels(ev *model.AlertEvent) map[string]string {
	if len(ev.Labels) == 0 {
		return nil
	}
	result := make(map[string]string)
	_ = json.Unmarshal(ev.Labels, &result)
	return result
}

// extractMatchers decodes a datatypes.JSON field into a []LabelMatcher slice.
func extractMatchers(raw interface{ MarshalJSON() ([]byte, error) }) []model.LabelMatcher {
	data, err := raw.MarshalJSON()
	if err != nil {
		return nil
	}
	var ms []model.LabelMatcher
	_ = json.Unmarshal(data, &ms)
	return ms
}

// StartRepeatTicker runs a 1-minute background loop that calls MaybeRepeat on every alert group.
// This ensures steady-state Firing alerts are re-notified once RepeatInterval has elapsed,
// even when no new state-change events are produced by the rule evaluator.
func (m *alertManager) StartRepeatTicker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixMilli()
			m.groupMu.Lock()
			for _, grp := range m.groups {
				grp.MaybeRepeat(now)
			}
			m.groupMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// CleanupGroups removes completed groups to prevent memory leaks.
func (m *alertManager) CleanupGroups(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.groupMu.Lock()
			for key, grp := range m.groups {
				if grp.isEmpty() {
					delete(m.groups, key)
				}
			}
			m.groupMu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}
