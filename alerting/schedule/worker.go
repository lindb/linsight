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
	"sync"
	"time"

	"github.com/lindb/common/pkg/logger"
	"go.uber.org/atomic"

	"github.com/lindb/linsight/model"
)

// RuleExecWorker executes a single alert rule on a goroutine using a lock-free channel design.
type RuleExecWorker interface {
	Run()
	UpdateRule(rule *model.AlertRule) bool
	EvalRule(tick int64) bool
	Stop()
}

type ruleExecWorker struct {
	ctx     context.Context
	cancel  context.CancelFunc
	deps    *Deps
	rawRule *model.AlertRule
	// two channels / one goroutine = lock free
	evalSignal chan int64
	updateCh   chan *model.AlertRule

	rule *AlertRule

	running   *atomic.Bool
	workerLog logger.Logger
}

func newRuleExecWorker(ctx context.Context, rule *model.AlertRule, deps *Deps) RuleExecWorker {
	c, cancel := context.WithCancel(ctx)
	return &ruleExecWorker{
		ctx:        c,
		cancel:     cancel,
		deps:       deps,
		rawRule:    rule,
		rule:       NewAlertRule(rule, deps),
		evalSignal: make(chan int64),
		updateCh:   make(chan *model.AlertRule),
		running:    atomic.NewBool(true),
		workerLog:  logger.GetLogger("Alerting", "RuleWorker"),
	}
}

func (w *ruleExecWorker) Run() {
	go pprof.Do(w.ctx, pprof.Labels("RuleWorker", w.rawRule.UID), func(ctx context.Context) {
		for {
			select {
			case tick := <-w.evalSignal:
				// skip paused or deleted rules
				if w.rawRule.State == model.RulePause || w.rawRule.State == model.RuleDelete {
					continue
				}
				// throttle: only execute on the appropriate tick multiple
				if tick%int64(w.rawRule.Interval.Seconds()/w.deps.Opts.Interval.Seconds()) != 0 {
					continue
				}
				if err := w.exec(); err != nil {
					w.workerLog.Error("rule exec failure",
						logger.String("rule", w.rawRule.UID),
						logger.Error(err))
				}
			case newRule := <-w.updateCh:
				// hot-reload rule config
				w.rawRule = newRule
				w.rule.Update(newRule)
			case <-ctx.Done():
				return
			}
		}
	})
}

func (w *ruleExecWorker) UpdateRule(rule *model.AlertRule) bool {
	select {
	case <-w.updateCh:
		// drain stale update
	default:
	}

	select {
	case w.updateCh <- rule:
		return true
	case <-w.ctx.Done():
		return false
	}
}

func (w *ruleExecWorker) EvalRule(tick int64) bool {
	select {
	case <-w.evalSignal:
		// drain stale signal
	default:
	}

	select {
	case w.evalSignal <- tick:
		return true
	case <-w.ctx.Done():
		return false
	}
}

func (w *ruleExecWorker) Stop() {
	if w.running.CompareAndSwap(true, false) {
		w.cancel()
	}
}

func (w *ruleExecWorker) exec() error {
	if err := w.rule.Perpare(); err != nil {
		return err
	}
	w.workerLog.Info("rule prepared",
		logger.String("rule", w.rawRule.UID),
		logger.Int32("version", int32(w.rawRule.Version)))

	// fetch metric data from datasource
	seriesList, err := w.rule.FetchData()
	if err != nil {
		return err
	}
	w.workerLog.Info("rule data fetched",
		logger.String("rule", w.rawRule.UID),
		logger.Int32("dataPoints", int32(len(seriesList))))

	// evaluate conditions and drive the state machine
	evalTime := time.Now()
	events, err := w.rule.EvalAndTrack(context.TODO(), seriesList, evalTime)
	if err != nil {
		return err
	}
	w.workerLog.Info("rule evaluated",
		logger.String("rule", w.rawRule.UID),
		logger.Int32("events", int32(len(events))),
		logger.Int32("active", int32(w.rule.ActiveAlerts())))
	for _, ev := range events {
		w.workerLog.Info("alert event",
			logger.String("rule", w.rawRule.UID),
			logger.String("fingerprint", ev.Fingerprint),
			logger.Int32("state", int32(ev.State)))
	}

	// delegate notification routing to the AlertManager
	if len(events) > 0 && w.deps.AlertMgr != nil {
		w.deps.AlertMgr.ProcessEvents(w.rawRule.OrgID, events)
	}
	return nil
}

// WorkerManager creates and tracks RuleExecWorkers per rule UID.
type WorkerManager interface {
	GetOrCreateWorker(ctx context.Context, rule *model.AlertRule) RuleExecWorker
}

type workerManager struct {
	orgID   int64
	deps    *Deps
	workers map[string]RuleExecWorker

	lock sync.RWMutex
}

// NewWorkerManager creates a WorkerManager.
func NewWorkerManager(orgID int64, deps *Deps) WorkerManager {
	return &workerManager{
		orgID:   orgID,
		deps:    deps,
		workers: make(map[string]RuleExecWorker),
	}
}

func (mgr *workerManager) GetOrCreateWorker(ctx context.Context, rule *model.AlertRule) RuleExecWorker {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	worker, ok := mgr.workers[rule.UID]
	if ok {
		return worker
	}

	// start a new worker goroutine for this rule
	worker = newRuleExecWorker(ctx, rule, mgr.deps)
	worker.Run()

	mgr.workers[rule.UID] = worker
	return worker
}
