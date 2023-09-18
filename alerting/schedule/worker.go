package schedule

import (
	"context"
	"fmt"
	"runtime/pprof"
	"sync"

	"go.uber.org/atomic"

	"github.com/lindb/linsight/model"
)

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
	// two chan/one goroutine = lock free
	evalSignal chan int64
	updateCh   chan *model.AlertRule

	rule *AlertRule

	running *atomic.Bool
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
	}
}

func (w *ruleExecWorker) Run() {
	go pprof.Do(w.ctx, pprof.Labels("RuleWorker", w.rawRule.UID), func(ctx context.Context) {
		for {
			select {
			case tick := <-w.evalSignal:
				// check rule state
				if w.rawRule.State == model.RulePause || w.rawRule.State == model.RuleDelete {
					continue
				}
				// check if need ready to exec
				if tick%int64(w.rawRule.Interval.Seconds()/w.deps.Opts.Interval.Seconds()) != 0 {
					continue
				}
				// TODO: time shuffle
				if err := w.exec(); err != nil {
					// TODO: update rule state
					fmt.Println(err)
				}
			case newRule := <-w.updateCh:
				// update alert rule
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
	//TODO: add metric
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
	//TODO: add metric
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
	if w.running.CAS(true, false) {
		w.cancel()
	}
}

func (w *ruleExecWorker) exec() error {
	err := w.rule.Perpare()
	if err != nil {
		return err
	}

	// fetch alert data
	seriesList, err := w.rule.FetchData()
	if err != nil {
		return err
	}

	// eval alert conditions
	alerts, err := w.rule.Eval(context.TODO(), seriesList)
	if err != nil {
		return err
	}

	// notifiy
	err = w.deps.notificationMgr.Notifiy(w.rule, alerts)
	if err != nil {
		return err
	}
	return nil
}

type WorkerManager interface {
	GetOrCreateWorker(ctx context.Context, rule *model.AlertRule) RuleExecWorker
}

type workerManager struct {
	orgID   int64
	deps    *Deps
	workers map[string]RuleExecWorker

	lock sync.RWMutex
}

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

	// run new rule exec worker
	worker = newRuleExecWorker(ctx, rule, mgr.deps)
	worker.Run()

	mgr.workers[rule.UID] = worker
	return worker
}
