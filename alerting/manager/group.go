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
	"sync"
	"time"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
)

// alertGroup holds a set of alerts that share the same routing label values.
// It schedules notifications according to GroupWait / GroupInterval / RepeatInterval.
type alertGroup struct {
	orgID    int64
	key      string
	route    *model.AlertRoute
	deps     *Deps

	events       map[string]*model.AlertEvent // keyed by fingerprint
	lastNotified int64                         // unix millis of last notification
	firstAdded   int64                         // unix millis when the group was first populated

	mu     sync.Mutex
	logger logger.Logger
}

func newAlertGroup(orgID int64, key string, route *model.AlertRoute, deps *Deps) *alertGroup {
	return &alertGroup{
		orgID:  orgID,
		key:    key,
		route:  route,
		deps:   deps,
		events: make(map[string]*model.AlertEvent),
		logger: logger.GetLogger("Alerting", "AlertGroup"),
	}
}

// addEvent incorporates a new or updated alert event into the group and triggers
// notification scheduling if the relevant time window has elapsed.
func (g *alertGroup) addEvent(ev *model.AlertEvent, now int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	isNew := g.firstAdded == 0
	if isNew {
		g.firstAdded = now
	}

	g.events[ev.Fingerprint] = ev

	// determine which interval applies
	var waitDuration time.Duration
	if isNew || g.lastNotified == 0 {
		waitDuration = g.groupWait()
	} else {
		waitDuration = g.groupInterval()
	}

	// fire notification if the window has passed
	if now-g.lastNotified >= waitDuration.Milliseconds() || (isNew && waitDuration == 0) {
		g.scheduleNotification(now)
	}
}

func (g *alertGroup) scheduleNotification(now int64) {
	events := g.allEvents()
	if len(events) == 0 {
		return
	}

	g.logger.Info("scheduling notification",
		logger.String("group", g.key),
		logger.Int64("org", g.orgID),
		logger.String("contactPointUID", g.route.ContactPointUID),
		logger.Int32("events", int32(len(events))))

	d := newDispatcher(g.deps, g.route.ContactPointUID, g.orgID)
	if err := d.dispatch(events, g.buildGroupLabels()); err != nil {
		g.logger.Error("dispatch notification failure",
			logger.String("group", g.key),
			logger.Int64("org", g.orgID),
			logger.Error(err))
		return
	}
	g.lastNotified = now

	// update NotifiedAt for all events
	for _, ev := range events {
		ev.NotifiedAt = now
		if err := g.deps.AlertEventSrv.UpsertAlertEvent(g.orgID, ev); err != nil {
			g.logger.Warn("update notified_at failure",
				logger.String("fingerprint", ev.Fingerprint),
				logger.Error(err))
		}
	}
}

// MaybeRepeat checks if RepeatInterval has elapsed since the last notification and re-dispatches.
func (g *alertGroup) MaybeRepeat(now int64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.lastNotified == 0 {
		return
	}
	if now-g.lastNotified >= g.repeatInterval().Milliseconds() {
		g.scheduleNotification(now)
	}
}

func (g *alertGroup) allEvents() []*model.AlertEvent {
	result := make([]*model.AlertEvent, 0, len(g.events))
	for _, ev := range g.events {
		result = append(result, ev)
	}
	return result
}

func (g *alertGroup) isEmpty() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, ev := range g.events {
		if ev.State == model.AlertEventFiring || ev.State == model.AlertEventPending {
			return false
		}
	}
	return true
}

func (g *alertGroup) buildGroupLabels() map[string]string {
	groupBy, _ := g.route.ToGroupBy()
	if len(groupBy) == 0 || len(g.events) == 0 {
		return nil
	}
	// use labels from an arbitrary event
	var sample *model.AlertEvent
	for _, ev := range g.events {
		sample = ev
		break
	}
	labels := parseEventLabels(sample)
	result := make(map[string]string, len(groupBy))
	for _, k := range groupBy {
		result[k] = labels[k]
	}
	return result
}

func (g *alertGroup) groupWait() time.Duration {
	if g.route.GroupWait > 0 {
		return time.Duration(g.route.GroupWait)
	}
	return time.Duration(model.DefaultPolicyGroupWait)
}

func (g *alertGroup) groupInterval() time.Duration {
	if g.route.GroupInterval > 0 {
		return time.Duration(g.route.GroupInterval)
	}
	return time.Duration(model.DefaultPolicyGroupInterval)
}

func (g *alertGroup) repeatInterval() time.Duration {
	if g.route.RepeatInterval > 0 {
		return time.Duration(g.route.RepeatInterval)
	}
	return time.Duration(model.DefaultPolicyRepeatInterval)
}
