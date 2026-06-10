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

	ljson "github.com/lindb/arrow/pkg/json"

	"github.com/lindb/linsight/model"
)

// Querier fetches raw datasource results for an alert rule's queries.
// It returns the Arrow-format ResultSet directly; callers are responsible
// for interpreting schema columns and row values.
type Querier interface {
	Query(queries []*model.Query, timerange model.TimeRange) (map[string]*ljson.ResultSet, error)
}

type querier struct {
	deps *Deps
	rule *model.AlertRule
}

func NewQuerier(rule *model.AlertRule, deps *Deps) Querier {
	return &querier{
		deps: deps,
		rule: rule,
	}
}

func (q *querier) Query(queries []*model.Query, timerange model.TimeRange) (map[string]*ljson.ResultSet, error) {
	ctx := context.TODO()
	rs := make(map[string]*ljson.ResultSet)
	for _, query := range queries {
		ds, err := q.deps.DatasourceSrv.GetDatasourceByOrgAndUID(q.rule.OrgID, query.Datasource.UID)
		if err != nil {
			return nil, err
		}
		cli, err := q.deps.DatasourceMgr.GetPlugin(ds)
		if err != nil {
			return nil, err
		}
		resp, err := cli.DataQuery(ctx, query, timerange)
		if err != nil {
			return nil, err
		}
		if resp != nil {
			rs[query.RefID] = resp
		}
	}
	return rs, nil
}
