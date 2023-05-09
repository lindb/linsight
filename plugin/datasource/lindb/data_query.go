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

package lindb

import (
	sq "github.com/Masterminds/squirrel"
)

func buildDataSQL(req *DataQueryRequest) (string, error) {
	groupBy := req.GroupBy
	if req.Stats {
		groupBy = append(groupBy, "time()")
	}
	// quoting fields: https://github.com/Masterminds/squirrel/issues/94
	builder := sq.Select(req.Fields...).
		From(req.Metric).
		GroupBy(groupBy...)
	sql, _, err := builder.ToSql()
	if err != nil {
		return "", err
	}
	return sql, nil
}
