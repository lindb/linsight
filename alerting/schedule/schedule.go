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
	"context"
	"runtime/pprof"
	"strconv"
	"time"

	"github.com/lindb/common/pkg/logger"
	"go.uber.org/atomic"

	"github.com/lindb/linsight/model"
)

type Options struct {
	Interval        time.Duration
	MinEvalInterval time.Duration
}

type Schedule interface {
	Run()
}

type schedule struct {
	ctx             context.Context
	orgID           int64
	deps            *Deps
	notificationMgr NotificationManager
	workerMgr       WorkerManager

	ruleCh      chan []model.AlertRule
	fetchSignal chan struct{}
	fetching    *atomic.Bool

	logger logger.Logger
}

func NewSchedule(ctx context.Context, orgID int64, deps *Deps) Schedule {
	s := &schedule{
		ctx:             ctx,
		orgID:           orgID,
		deps:            deps,
		notificationMgr: NewNotificationManager(orgID, deps),
		workerMgr:       NewWorkerManager(orgID, deps),

		ruleCh:      make(chan []model.AlertRule),
		fetchSignal: make(chan struct{}),
		fetching:    atomic.NewBool(false),

		logger: logger.GetLogger("Alering", "Schedule"),
	}
	s.deps.notificationMgr = s.notificationMgr
	return s
}

func (s *schedule) Run() {
	// run rule scheduler
	s.runScheduler()
	// run config fetcher
	s.runFetcher()
}

func (s *schedule) runScheduler() {
	ticker := time.NewTicker(s.deps.Opts.Interval)

	go pprof.Do(s.ctx, pprof.Labels("Scheduler", strconv.Itoa(int(s.orgID))), func(ctx context.Context) {
		for {
			select {
			case tick := <-ticker.C:
				if s.fetching.CAS(false, true) {
					// notifiy fetch config from storage(alert ruel/notifications etc.)
					s.fetchSignal <- struct{}{}
				}
				s.processRules(tick)
			case <-ctx.Done():
				ticker.Stop()
				if err := ctx.Err(); err != nil {
					s.logger.Error("alerting schedule exist", logger.Int64("org", s.orgID), logger.Error(err))
				}
				return
			}
		}
	})
}

func (s *schedule) runFetcher() {
	go pprof.Do(s.ctx, pprof.Labels("ConfigFetcher", strconv.Itoa(int(s.orgID))), func(ctx context.Context) {
		for {
			select {
			case <-s.fetchSignal:
				fetch := func() {
					defer s.fetching.Store(false)
					// load global notification settings by org
					s.notificationMgr.LoadNotificationSettings()
					// process alert rules by org
					s.loadAlertRules()
				}

				// fetch config
				fetch()
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					s.logger.Error("alerting config fetch exist", logger.Int64("org", s.orgID), logger.Error(err))
				}
				return
			}
		}
	})

}

func (s *schedule) loadAlertRules() {
	rules, err := s.deps.AlertRuleSrv.FetchAllAlertRules(&model.SearchAlertRuleRequest{})
	if err != nil {
		s.logger.Error("fetch alert rule failure", logger.Any("org", s.orgID), logger.Error(err))
		return
	}

	// Reload config for active orgs and purge state for orgs with no remaining rules.
	// Runs before rules are dispatched to workers so config is always fresh when
	// the first evaluation happens — this also handles startup pre-warming.
	if s.deps.AlertMgr != nil {
		orgSet := make(map[int64]struct{})
		for i := range rules {
			orgSet[rules[i].OrgID] = struct{}{}
		}
		s.deps.AlertMgr.ReconcileOrgs(orgSet)
	}

	s.ruleCh <- rules
}

func (s *schedule) processRules(tickTime time.Time) {
	select {
	case rules := <-s.ruleCh:
		tick := tickTime.Unix() / int64(s.deps.Opts.Interval.Seconds())
		activeUIDs := make(map[string]struct{}, len(rules))
		for idx := range rules {
			s.processRule(&rules[idx], tick)
			activeUIDs[rules[idx].UID] = struct{}{}
		}
		// Stop and remove workers for rules that no longer exist in the DB.
		s.workerMgr.ReconcileWorkers(activeUIDs)
	case <-s.ctx.Done():
		return
	}
}

func (s *schedule) processRule(rule *model.AlertRule, tick int64) {
	// if rule eval interval < min eval interval, need reset rule eval interval
	if rule.Interval < model.Duration(s.deps.Opts.MinEvalInterval) {
		rule.Interval = model.Duration(s.deps.Opts.MinEvalInterval)
	}

	worker := s.workerMgr.GetOrCreateWorker(s.ctx, rule)

	// try update rule config
	worker.UpdateRule(rule)
	// try trigger rule eval
	worker.EvalRule(tick)
}

func getValue(values map[int64]float64) (int64, float64) {
	for t, v := range values {
		return t, v
	}
	return 0, 0
}
