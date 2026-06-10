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

package schedule

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"
	"time"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// RuleStateManager tracks alert event state for every fingerprint in a rule evaluation cycle.
// It drives the Pending → Firing → Resolved state machine.
type RuleStateManager interface {
	// Eval processes the current matching series for a rule and returns all state-changed AlertEvents.
	Eval(rule *model.AlertRule, matchingSeries []*model.Series, evalTime time.Time) []*model.AlertEvent
	// ActiveCount returns the number of fingerprints currently in Pending or Firing state.
	ActiveCount() int
}

// fingerprintState holds persistent state for a single alert fingerprint.
type fingerprintState struct {
	event        *model.AlertEvent
	seenLastEval bool // true when this fingerprint appeared in the most recent evaluation
}

type ruleStateManager struct {
	states map[string]*fingerprintState // key: fingerprint
	mu     sync.Mutex
	logger logger.Logger
}

// NewRuleStateManager creates a RuleStateManager instance.
func NewRuleStateManager() RuleStateManager {
	return &ruleStateManager{
		states: make(map[string]*fingerprintState),
		logger: logger.GetLogger("Alerting", "RuleState"),
	}
}

// Eval runs the state machine for a single evaluation cycle.
// matchingSeries contains the series that satisfied at least one alert condition.
func (m *ruleStateManager) Eval(rule *model.AlertRule, matchingSeries []*model.Series, evalTime time.Time) []*model.AlertEvent {
	nowMs := evalTime.UnixMilli()
	var changed []*model.AlertEvent

	m.mu.Lock()
	defer m.mu.Unlock()

	// mark all existing states as unseen for this round
	for _, st := range m.states {
		st.seenLastEval = false
	}

	for _, series := range matchingSeries {
		labels := buildLabels(rule, series)
		fp := computeFingerprint(labels)

		st, exists := m.states[fp]
		if !exists {
			// brand new fingerprint — create Pending event
			ev := &model.AlertEvent{
				RuleUID:      rule.UID,
				RuleTitle:    rule.Title,
				Fingerprint:  fp,
				Severity:     series.Severity,
				Labels:       marshalLabels(labels),
				Annotations:  renderAnnotations(rule, series),
				Value:        series.Value,
				PendingSince: nowMs,
				GeneratorURL: fmt.Sprintf("/explore?ruleUID=%s", rule.UID),
			}

			if rule.For == 0 {
				// no 'for' duration — fire immediately
				ev.State = model.AlertEventFiring
				ev.StartsAt = nowMs
			} else {
				ev.State = model.AlertEventPending
			}
			st = &fingerprintState{event: ev}
			m.states[fp] = st
			changed = append(changed, ev)
			m.logger.Info("alert state transition",
				logger.String("rule", rule.UID),
				logger.String("fingerprint", fp),
				logger.String("transition", "new→"+stateLabel(ev.State)),
				logger.Int32("severity", int32(ev.Severity)))
		} else {
			ev := st.event
			ev.Value = series.Value
			ev.Severity = series.Severity // severity may change if the rule is updated
			ev.Annotations = renderAnnotations(rule, series)

			switch ev.State {
			case model.AlertEventPending:
				// check if 'for' duration has elapsed
				if rule.For > 0 && nowMs-ev.PendingSince >= rule.For.Milliseconds() {
					ev.State = model.AlertEventFiring
					ev.StartsAt = nowMs
					changed = append(changed, ev)
					m.logger.Info("alert state transition",
						logger.String("rule", rule.UID),
						logger.String("fingerprint", fp),
						logger.String("transition", "pending→firing"),
						logger.Int32("severity", int32(ev.Severity)))
				}
			case model.AlertEventResolved:
				// condition is true again after being resolved — re-open
				ev.State = model.AlertEventPending
				ev.PendingSince = nowMs
				ev.StartsAt = 0
				ev.EndsAt = 0
				if rule.For == 0 {
					ev.State = model.AlertEventFiring
					ev.StartsAt = nowMs
				}
				changed = append(changed, ev)
				m.logger.Info("alert state transition",
					logger.String("rule", rule.UID),
					logger.String("fingerprint", fp),
					logger.String("transition", "resolved→"+stateLabel(ev.State)),
					logger.Int32("severity", int32(ev.Severity)))
			}
		}
		st.seenLastEval = true
	}

	// handle disappeared fingerprints — transition to Resolved
	for fp, st := range m.states {
		if st.seenLastEval {
			continue
		}
		ev := st.event
		if ev.State == model.AlertEventFiring || ev.State == model.AlertEventPending {
			ev.State = model.AlertEventResolved
			ev.EndsAt = nowMs
			changed = append(changed, ev)
			m.logger.Info("alert state transition",
				logger.String("rule", rule.UID),
				logger.String("fingerprint", fp),
				logger.String("transition", "firing→resolved"),
				logger.Int32("severity", int32(ev.Severity)))
		}
		// clean up resolved states after recording
		delete(m.states, fp)
	}

	m.logger.Info("state machine evaluated",
		logger.String("rule", rule.UID),
		logger.Int32("matched", int32(len(matchingSeries))),
		logger.Int32("changed", int32(len(changed))),
		logger.Int32("tracked", int32(len(m.states))))

	return changed
}

// buildLabels merges rule-level labels, series tags and system labels into a single map.
func buildLabels(rule *model.AlertRule, series *model.Series) map[string]string {
	labels := make(map[string]string)

	// 1. rule static labels (lowest priority)
	if len(rule.Labels) > 0 {
		_ = json.Unmarshal(rule.Labels, &labels)
	}

	// 2. series tags
	for k, v := range series.Tags {
		labels[k] = v
	}

	// 3. system labels (highest priority)
	labels["alertname"] = rule.Title
	labels["__rule_uid__"] = rule.UID
	labels["__severity__"] = fmt.Sprintf("%d", series.Severity) // include severity so different levels get distinct fingerprints
	if series.MetricName != "" {
		labels["__metric__"] = series.MetricName
	}

	return labels
}

// renderAnnotations evaluates rule annotation templates with the current series context.
// The resulting JSON is stored on the AlertEvent.
func renderAnnotations(rule *model.AlertRule, series *model.Series) []byte {
	if len(rule.Annotations) == 0 {
		return nil
	}
	// annotations are a map[string]string where values may be plain strings or
	// Go template strings; for now we store them verbatim and rendering happens
	// in the template layer.
	return rule.Annotations
}

// computeFingerprint returns a stable hash of a sorted label set (FNV-64a, same as Prometheus).
func computeFingerprint(labels map[string]string) string {
	// sort keys for deterministic output
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := fnv.New64a()
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte("="))
		_, _ = h.Write([]byte(labels[k]))
		_, _ = h.Write([]byte("\x00"))
	}
	return fmt.Sprintf("%016x", h.Sum64())
}

// marshalLabels converts a string map to a JSON byte slice.
func marshalLabels(labels map[string]string) []byte {
	data, _ := json.Marshal(labels)
	return data
}

// ActiveCount returns the number of fingerprints currently in Pending or Firing state.
func (m *ruleStateManager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, st := range m.states {
		if st.event.State == model.AlertEventFiring || st.event.State == model.AlertEventPending {
			n++
		}
	}
	return n
}

// stateLabel returns a human-readable name for an AlertEventState.
func stateLabel(s model.AlertEventState) string {
	switch s {
	case model.AlertEventPending:
		return "pending"
	case model.AlertEventFiring:
		return "firing"
	case model.AlertEventResolved:
		return "resolved"
	default:
		return fmt.Sprintf("state(%d)", s)
	}
}
