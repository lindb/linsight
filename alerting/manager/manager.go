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
	ExternalURL             string
}

type alertManager struct {
	deps    *Deps
	groups  map[string]*alertGroup // key: orgID:receiverUID:groupKey
	groupMu sync.Mutex

	// cached config per org
	orgRoutes   map[int64][]*model.AlertRoute
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
		orgRoutes:   make(map[int64][]*model.AlertRoute),
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

	silences, inhibits, routes := m.getOrgConfig(orgID)

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
		matched := m.matchRoutes(ev, routes)
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
				logger.String("receiverUID", route.ReceiverUID))
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
	routes, err := m.deps.AlertRouteSrv.GetAlertRoutesByOrg(orgID)
	if err != nil {
		m.logger.Error("reload routes failure", logger.Int64("org", orgID), logger.Error(err))
		routes = nil
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
	m.orgRoutes[orgID] = routes
	m.orgSilences[orgID] = silences
	m.orgInhibits[orgID] = inhibits
	m.cfgMu.Unlock()
}

// getOrgConfig returns a snapshot of the current configuration for an org.
func (m *alertManager) getOrgConfig(orgID int64) ([]*model.AlertSilence, []*model.AlertInhibitRule, []*model.AlertRoute) {
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

// matchRoutes returns all routes that match the given event.
func (m *alertManager) matchRoutes(ev *model.AlertEvent, routes []*model.AlertRoute) []*model.AlertRoute {
	labels := parseEventLabels(ev)
	var matched []*model.AlertRoute

	for _, route := range routes {
		matchers, err := route.ToMatchers()
		if err != nil {
			continue
		}
		if matchAll(labels, matchers) {
			matched = append(matched, route)
			if !route.Continue {
				break
			}
		}
	}
	return matched
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
