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
	s.ruleCh <- rules
}

func (s *schedule) processRules(tickTime time.Time) {
	select {
	case rules := <-s.ruleCh:
		tick := tickTime.Unix() / int64(s.deps.Opts.Interval.Seconds())
		for idx := range rules {
			s.processRule(&rules[idx], tick)
		}
	case <-s.ctx.Done():
		return
	}
}

func (s *schedule) processRule(rule *model.AlertRule, tick int64) {
	// if rule eval interval < min eval interval, need reset rule eval interval
	if rule.Interval < s.deps.Opts.MinEvalInterval {
		rule.Interval = s.deps.Opts.MinEvalInterval
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
