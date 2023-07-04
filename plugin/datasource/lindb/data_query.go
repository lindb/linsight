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
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/lindb/linsight/model"
)

// buildDataQuerySQL builds LinDB query language based on data query request and time range.
func buildDataQuerySQL(req *DataQueryRequest, timeRange model.TimeRange) (string, error) {
	groupBy := req.GroupBy
	if req.Stats {
		groupBy = append(groupBy, "time()")
	}
	builder := New().Select(req.Fields...).
		Metric(req.Metric).
		Namespace(req.Namespace).
		Where(req.Where...).
		GroupBy(groupBy...)

	builder.TimeRange(timeRange)
	sql, err := builder.ToSQL()
	if err != nil {
		return "", err
	}
	return sql, nil
}

// DataQueryBuilder represents LinDB query language builder.
type DataQueryBuilder struct {
	fields    []string
	metric    string
	namespace string
	groupBy   []string
	where     []Expr
}

// New creates a builder.
func New() *DataQueryBuilder {
	return &DataQueryBuilder{}
}

// Select sets field list of query or functions.
func (b *DataQueryBuilder) Select(fields ...string) *DataQueryBuilder {
	b.fields = append(b.fields, fields...)
	return b
}

// Metric sets metric name.
func (b *DataQueryBuilder) Metric(metric string) *DataQueryBuilder {
	b.metric = metric
	return b
}

// Namespace sets namespace.
func (b *DataQueryBuilder) Namespace(namespace string) *DataQueryBuilder {
	b.namespace = namespace
	return b
}

// Where sets where conditions.
func (b *DataQueryBuilder) Where(where ...Expr) *DataQueryBuilder {
	for _, e := range where {
		if e.Key == "" || e.Op == "" {
			// ignore key/op empty
			continue
		}
		b.where = append(b.where, e)
	}
	return b
}

// GroupBy sets group by.
func (b *DataQueryBuilder) GroupBy(groupBy ...string) *DataQueryBuilder {
	b.groupBy = append(b.groupBy, groupBy...)
	return b
}

// TimeRange sets time range.
func (b *DataQueryBuilder) TimeRange(timeRange model.TimeRange) *DataQueryBuilder {
	b.setTimeCondition(timeRange.From, GtEq)
	b.setTimeCondition(timeRange.To, LtEq)
	return b
}

// ToSQL returns the LinDB query language based on params.
func (b *DataQueryBuilder) ToSQL() (sql string, err error) {
	if len(b.fields) == 0 {
		err = fmt.Errorf("select statements must have at least one field")
		return
	}
	if b.metric == "" {
		err = fmt.Errorf("metric name is required")
		return
	}

	sqlBuf := &bytes.Buffer{}
	sqlBuf.WriteString("SELECT ")

	sqlBuf.WriteString(strings.Join(b.fields, ","))
	sqlBuf.WriteString(" FROM ")
	fmt.Fprintf(sqlBuf, "'%s'", b.metric)

	if len(b.namespace) > 0 {
		sqlBuf.WriteString(" ON ")
		fmt.Fprintf(sqlBuf, "'%s'", b.namespace)
	}

	if len(b.where) > 0 {
		sqlBuf.WriteString(" WHERE ")
		sqlBuf.WriteString(b.joinWhere(" AND "))
	}

	if len(b.groupBy) > 0 {
		sqlBuf.WriteString(" GROUP BY ")
		sqlBuf.WriteString(strings.Join(b.groupBy, ","))
	}

	return sqlBuf.String(), nil
}

// joinWhere builds where conditions.
func (b *DataQueryBuilder) joinWhere(sep string) string {
	if len(b.where) == 1 {
		return b.where[0].String()
	}

	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "%s", b.where[0].String())
	for i := 1; i < len(b.where); i++ {
		buffer.WriteString(sep)
		fmt.Fprintf(buffer, "%s", b.where[i].String())
	}
	return buffer.String()
}

// isTimestamp checks if timestamp format.
func (b *DataQueryBuilder) isTimestamp(timestamp string) bool {
	_, err := time.Parse("2006-01-02 15:04:05", timestamp)
	return err == nil
}

// setTimeCondition set timestamp condition
func (b *DataQueryBuilder) setTimeCondition(timestamp string, op Operator) {
	if timestamp != "" {
		if b.isTimestamp(timestamp) {
			b.Where(Expr{Key: "time", Op: op, Value: timestamp})
		} else {
			b.Where(Expr{Key: "time", Op: op, Value: timestamp, raw: true})
		}
	}
}
