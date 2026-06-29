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
	// ReconcileOrgs reloads config for orgs with active rules and purges all cached state
	// (config maps and alert groups) for orgs that no longer have any rules.
	ReconcileOrgs(activeOrgIDs map[int64]struct{})
	// StartRepeatTicker runs a background loop that periodically calls MaybeRepeat on all
	// alert groups so that steady-state Firing alerts get re-notified after RepeatInterval.
	StartRepeatTicker(ctx context.Context)
	// CleanupGroups runs a background loop that periodically removes idle alert groups
	// to prevent unbounded memory growth.
	CleanupGroups(ctx context.Context)
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

	silences, inhibits, root := m.getOrgConfig(orgID)

	// fetch DB-persisted firing events so InhibitStage can detect cross-batch source alerts
	dbFiring, _ := m.deps.AlertEventSrv.GetFiringEventsByOrg(orgID)

	pipeline := Pipeline{
		&SilenceStage{silences: silences, now: now, logger: m.logger},
		&InhibitStage{inhibits: inhibits, allFiring: mergeFiring(dbFiring, events), logger: m.logger},
		&RouteStage{root: root, mgr: m, logger: m.logger},
	}
	pipeline.Run(&stageCtx{orgID: orgID, now: now, deps: m.deps}, events)
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

// ReconcileOrgs reloads config for active orgs and purges all cached state for orgs
// that no longer have any rules (config maps + in-flight alert groups).
func (m *alertManager) ReconcileOrgs(activeOrgIDs map[int64]struct{}) {
	// Refresh route/silence/inhibit config for every org that still has active rules.
	for orgID := range activeOrgIDs {
		m.ReloadConfig(orgID)
	}

	// Remove cached config for orgs whose last rule has been deleted.
	m.cfgMu.Lock()
	for orgID := range m.orgRoutes {
		if _, ok := activeOrgIDs[orgID]; !ok {
			delete(m.orgRoutes, orgID)
			delete(m.orgSilences, orgID)
			delete(m.orgInhibits, orgID)
		}
	}
	m.cfgMu.Unlock()

	// Remove alert groups that belong to now-inactive orgs.
	m.groupMu.Lock()
	for key, grp := range m.groups {
		if _, ok := activeOrgIDs[grp.orgID]; !ok {
			delete(m.groups, key)
		}
	}
	m.groupMu.Unlock()
}
