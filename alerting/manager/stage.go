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
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// stageCtx carries per-call shared context across all pipeline stages.
type stageCtx struct {
	orgID int64
	now   int64
	deps  *Deps
}

// Stage processes a batch of alert events and returns the surviving events.
// Filter stages (Silence, Inhibit) drop events they suppress.
// The terminal stage (Route) dispatches events as a side effect and returns all events unchanged.
type Stage interface {
	Run(ctx *stageCtx, events []*model.AlertEvent) []*model.AlertEvent
}

// Pipeline is an ordered sequence of stages.
type Pipeline []Stage

// Run executes all stages in order; stops early when the event set becomes empty.
func (p Pipeline) Run(ctx *stageCtx, events []*model.AlertEvent) {
	for _, stage := range p {
		if len(events) == 0 {
			return
		}
		events = stage.Run(ctx, events)
	}
}

// ─── SilenceStage ────────────────────────────────────────────────────────────

// SilenceStage filters out events that match any active silence window.
type SilenceStage struct {
	silences []*model.AlertSilence
	now      int64
	logger   logger.Logger
}

func (s *SilenceStage) Run(_ *stageCtx, events []*model.AlertEvent) []*model.AlertEvent {
	out := events[:0]
	for _, ev := range events {
		if !s.isSilenced(ev) {
			out = append(out, ev)
		}
	}
	return out
}

func (s *SilenceStage) isSilenced(ev *model.AlertEvent) bool {
	labels := parseEventLabels(ev)
	for _, silence := range s.silences {
		if silence.StartsAt > s.now || silence.EndsAt <= s.now {
			continue
		}
		matchers, err := silence.ToMatchers()
		if err != nil {
			continue
		}
		if matchAll(labels, matchers) {
			s.logger.Info("alert event silenced",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID),
				logger.String("silenceUID", silence.UID))
			return true
		}
	}
	return false
}

// ─── InhibitStage ─────────────────────────────────────────────────────────────

// InhibitStage filters out events that are suppressed by any enabled inhibit rule.
//
// allFiring is the union of DB-persisted firing events and the current batch so that
// source alerts from previous evaluation cycles are not missed (P0 cross-batch fix).
type InhibitStage struct {
	inhibits  []*model.AlertInhibitRule
	allFiring []*model.AlertEvent
	logger    logger.Logger
}

func (s *InhibitStage) Run(_ *stageCtx, events []*model.AlertEvent) []*model.AlertEvent {
	out := events[:0]
	for _, ev := range events {
		if !s.isInhibited(ev) {
			out = append(out, ev)
		}
	}
	return out
}

func (s *InhibitStage) isInhibited(ev *model.AlertEvent) bool {
	targetLabels := parseEventLabels(ev)

	for _, rule := range s.inhibits {
		targetMatchers, err := rule.ToTargetMatchers()
		if err != nil {
			continue
		}
		// check if this event is a target of the inhibit rule
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
		for _, src := range s.allFiring {
			if src.Fingerprint == ev.Fingerprint {
				continue
			}
			srcLabels := parseEventLabels(src)
			if !matchAll(srcLabels, sourceMatchers) {
				continue
			}
			// all equal-label keys must share the same value between source and target
			inhibit := true
			for _, lbl := range equalLabels {
				if srcLabels[lbl] != targetLabels[lbl] {
					inhibit = false
					break
				}
			}
			if inhibit {
				s.logger.Info("alert event inhibited",
					logger.String("fingerprint", ev.Fingerprint),
					logger.String("rule", ev.RuleUID),
					logger.String("inhibitRuleUID", rule.UID))
				return true
			}
		}
	}
	return false
}

// ─── RouteStage ───────────────────────────────────────────────────────────────

// RouteStage is the terminal stage: it routes each event to the matching alert group
// for notification scheduling. Events are returned unchanged; dispatch is a side effect.
type RouteStage struct {
	root   *model.AlertRouteNode
	mgr    *alertManager
	logger logger.Logger
}

func (s *RouteStage) Run(ctx *stageCtx, events []*model.AlertEvent) []*model.AlertEvent {
	for _, ev := range events {
		if s.root == nil {
			s.logger.Info("no route tree configured",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID))
			continue
		}
		labels := parseEventLabels(ev)
		matched := traverseNode(s.root, labels, nil)
		if len(matched) == 0 {
			s.logger.Info("no route matched",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID))
			continue
		}
		for _, route := range matched {
			s.logger.Info("alert event routed",
				logger.String("fingerprint", ev.Fingerprint),
				logger.String("rule", ev.RuleUID),
				logger.String("routeUID", route.UID),
				logger.String("contactPointUID", route.ContactPointUID))
			groupKey := buildGroupKey(ctx.orgID, route, ev)
			s.mgr.groupMu.Lock()
			grp, ok := s.mgr.groups[groupKey]
			if !ok {
				grp = newAlertGroup(ctx.orgID, groupKey, route, ctx.deps)
				s.mgr.groups[groupKey] = grp
			}
			s.mgr.groupMu.Unlock()
			grp.addEvent(ev, ctx.now)
		}
	}
	return events
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// buildGroupKey constructs the unique key for an alert group.
func buildGroupKey(orgID int64, route *model.AlertRoute, ev *model.AlertEvent) string {
	labels := parseEventLabels(ev)
	groupBy, _ := route.ToGroupBy()
	groupValues := make([]byte, 0, 64)
	for _, k := range groupBy {
		groupValues = append(groupValues, k+"="+labels[k]+";"...)
	}
	return string(groupValues) + ":" + route.UID
}

// mergeFiring returns the union of DB-persisted firing events and the current batch,
// deduplicating by fingerprint. Batch events take precedence (they carry the freshest state).
func mergeFiring(dbFiring []*model.AlertEvent, batch []*model.AlertEvent) []*model.AlertEvent {
	seen := make(map[string]struct{}, len(batch))
	result := make([]*model.AlertEvent, 0, len(dbFiring)+len(batch))
	for _, ev := range batch {
		if ev.State == model.AlertEventFiring {
			seen[ev.Fingerprint] = struct{}{}
			result = append(result, ev)
		}
	}
	for _, ev := range dbFiring {
		if _, ok := seen[ev.Fingerprint]; !ok {
			result = append(result, ev)
		}
	}
	return result
}
