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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDataQuery_buildSQL(t *testing.T) {
	sql, err := buildDataQuerySQL(&DataQueryRequest{
		Stats:     true,
		Metric:    "system.host.cpu",
		Namespace: "ns",
		Fields:    []string{"load", "usage"},
		GroupBy:   []string{"host", "region"},
	}, "from", "2022-04-03 02:34:33")
	assert.NoError(t, err)
	assert.Equal(t, "SELECT load,usage FROM 'system.host.cpu' ON 'ns' "+
		"WHERE time >= 'from' AND time <= '2022-04-03 02:34:33' GROUP BY host,region,time()", sql)

	sql, err = buildDataQuerySQL(&DataQueryRequest{
		Stats:   true,
		Where:   []Expr{{}},
		Metric:  "system.host.cpu",
		Fields:  []string{"load", "usage"},
		GroupBy: []string{"host", "region"},
	}, "from", "2022-04-03 02:34:33")
	assert.NoError(t, err)
	assert.Equal(t, "SELECT load,usage FROM 'system.host.cpu' "+
		"WHERE time >= 'from' AND time <= '2022-04-03 02:34:33' GROUP BY host,region,time()", sql)

	sql, err = buildDataQuerySQL(&DataQueryRequest{
		Where:  []Expr{{Key: "key", Op: Eq, Value: "value"}},
		Metric: "system.host.cpu",
		Fields: []string{"load", "usage"},
	}, "", "")
	assert.NoError(t, err)
	assert.Equal(t, "SELECT load,usage FROM 'system.host.cpu' WHERE key = 'value'", sql)
}

func TestDataQuery_buildSQL_Failure(t *testing.T) {
	sql, err := buildDataQuerySQL(&DataQueryRequest{
		Metric: "system.host.cpu",
	}, "from", "2022-04-03 02:34:33")
	assert.Error(t, err)
	assert.Empty(t, sql)

	sql, err = buildDataQuerySQL(&DataQueryRequest{
		Fields: []string{"load", "usage"},
	}, "from", "2022-04-03 02:34:33")
	assert.Error(t, err)
	assert.Empty(t, sql)
}
