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
	"fmt"

	"github.com/lindb/common/models"

	"github.com/lindb/linsight/model"
)

type Querier interface {
	Query(queries []*model.Query, timerange model.TimeRange) (map[string]*models.ResultSet, error)
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

func (q *querier) Query(queries []*model.Query, timerange model.TimeRange) (map[string]*models.ResultSet, error) {
	ctx := context.TODO()
	rs := make(map[string]*models.ResultSet)
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
		// fmt.Println(string(encoding.JSONMarshal(resp)))

		// TODO: add refID maybe empty,go?
		resultset, ok := resp.(*models.ResultSet)
		if ok {
			rs[query.RefID] = resultset
		} else {
			fmt.Println("wrong result set type")
		}
	}
	return rs, nil
}
