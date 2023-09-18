package schedule

import (
	"context"
	"fmt"
	"sort"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/model"
)

type AlertRule struct {
	conditions    []*vm.Program
	queryParams   *model.QueryRequest
	notifications []string
	rule          *model.AlertRule
	querier       Querier

	deps           *Deps
	currentVersion int
}

func NewAlertRule(rule *model.AlertRule, deps *Deps) *AlertRule {
	return &AlertRule{
		rule:           rule,
		deps:           deps,
		querier:        NewQuerier(rule, deps),
		currentVersion: -1,
	}
}

func (r *AlertRule) Update(rule *model.AlertRule) {
	r.rule = rule
}

func (r *AlertRule) Perpare() error {
	rule := r.rule
	if r.currentVersion == rule.Version {
		// same version
		return nil
	}

	conditions, err := rule.ToConditions()
	if err != nil {
		return err
	}
	notifications, err := rule.ToNotificatins()
	if err != nil {
		return err
	}
	dataQuery, err := rule.ToDataQuery()
	if err != nil {
		return err
	}
	sort.Slice(conditions, func(i, j int) bool {
		return conditions[i].Severity < conditions[j].Severity
	})

	r.conditions = make([]*vm.Program, len(conditions))
	r.queryParams = dataQuery
	r.notifications = notifications

	for idx := range conditions {
		condition := conditions[idx]
		var str string
		if err := encoding.JSONUnmarshal(condition.Expr, &str); err != nil {
			return err
		}
		fmt.Println(str)
		program, err := expr.Compile(str)
		if err != nil {
			return err
		}
		r.conditions[idx] = program
	}
	return nil
}

func (r *AlertRule) GetNotifications() []string {
	return r.notifications
}

func (r *AlertRule) FetchData() (rs []*model.Series, err error) {
	resp, err := r.querier.Query(r.queryParams.Queries, r.queryParams.Range)
	if err != nil {
		return nil, err
	}

	for refID, resultSet := range resp {
		for _, s := range resultSet.Series {
			for fn, f := range s.Fields {
				ts, val := getValue(f)
				rs = append(rs, &model.Series{
					RefID:      refID,
					MetricName: resultSet.MetricName,
					Field:      fn,
					Tags:       s.Tags,
					Timestamp:  ts,
					Value:      val,
				})
			}
		}
	}
	return
}

func (r *AlertRule) Eval(ctx context.Context, seriesList []*model.Series) (rs []*model.Alert, err error) {
	for i := range seriesList {
		series := seriesList[i]
		env := series.ToMap()
		fmt.Println(string(encoding.JSONMarshal(series)))
		for idx := range r.conditions {
			fmt.Println(idx)
			output, err := expr.Run(r.conditions[idx], env)
			if err != nil {
				return nil, err
			}
			fmt.Println(output)
			result, ok := output.(bool)
			if ok && result {
				rs = append(rs, &model.Alert{
					Series: series,
				})
			}
		}
	}
	return rs, nil
}
