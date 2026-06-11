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
	"testing"

	ljson "github.com/lindb/arrow/pkg/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
)

// ----------------------------------------------------------------------------
// Test doubles
// ----------------------------------------------------------------------------

// stubQuerier returns a fixed ResultSet map; safe for parallel tests.
type stubQuerier struct {
	result map[string]*ljson.ResultSet
}

func (s *stubQuerier) Query(_ []*model.Query, _ model.TimeRange) (map[string]*ljson.ResultSet, error) {
	return s.result, nil
}

// captureQuerier invokes a callback so tests can inspect the arguments passed to Query.
type captureQuerier struct {
	onQuery func([]*model.Query, model.TimeRange) (map[string]*ljson.ResultSet, error)
}

func (c *captureQuerier) Query(q []*model.Query, tr model.TimeRange) (map[string]*ljson.ResultSet, error) {
	return c.onQuery(q, tr)
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// dataJSON with an explicit numeric Range so tests get deterministic timestamps.
const testDataJSON = `{"range":{"start":1000,"end":2000},"queries":[{"refId":"A","datasource":{"uid":"ds1"},"params":{}}]}`

// newTestRule builds an AlertRule with empty (passthrough) conditions and the given querier.
// Perpare() is called before returning so queryParams is populated.
func newTestRule(t *testing.T, dataJSON string, q Querier) *AlertRule {
	t.Helper()
	rule := &model.AlertRule{
		UID:        "test-rule",
		Version:    1,
		Data:       datatypes.JSON(dataJSON),
		Conditions: datatypes.JSON(`[]`), // empty → passthrough; no compilation needed
	}
	ar := NewAlertRule(rule, &Deps{})
	ar.querier = q
	require.NoError(t, ar.Perpare())
	return ar
}

// ----------------------------------------------------------------------------
// FetchData tests
// ----------------------------------------------------------------------------

// TestFetchData_ScalarColumnsMerged verifies that all numeric columns in a single row
// are merged into one DataPoint.Values map instead of producing multiple DataPoints.
func TestFetchData_ScalarColumnsMerged(t *testing.T) {
	rs := &ljson.ResultSet{
		Schema: &ljson.Schema{
			Columns: []ljson.ColumnMetadata{
				{Name: "host", DataType: "string"},
				{Name: "cpu_usage", DataType: "float64"},
				{Name: "mem_usage", DataType: "float64"},
			},
		},
		Rows: [][]any{
			{"server1", float64(75.0), float64(60.0)},
		},
	}
	ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

	dps, err := ar.FetchData()
	require.NoError(t, err)
	// One row with two scalar cols → exactly one DataPoint (not two)
	require.Len(t, dps, 1)

	dp := dps[0]
	assert.Equal(t, "A", dp.RefID)
	assert.Equal(t, map[string]string{"host": "server1"}, dp.Labels)
	assert.Equal(t, float64(75.0), dp.Values["cpu_usage"])
	assert.Equal(t, float64(60.0), dp.Values["mem_usage"])
	// scalar DataPoint timestamp must equal Range.End (not Start, not 0)
	assert.Equal(t, int64(2000), dp.Timestamp)
}

// TestFetchData_RepresentativeValue verifies the priority rule for DataPoint.Value:
// "value" column is preferred; otherwise the first numeric column in schema order is used.
func TestFetchData_RepresentativeValue(t *testing.T) {
	t.Run("value_column_wins", func(t *testing.T) {
		rs := &ljson.ResultSet{
			Schema: &ljson.Schema{
				Columns: []ljson.ColumnMetadata{
					{Name: "cpu_usage", DataType: "float64"},
					{Name: "value", DataType: "float64"}, // named "value" → wins
				},
			},
			Rows: [][]any{{float64(75.0), float64(99.0)}},
		}
		ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

		dps, err := ar.FetchData()
		require.NoError(t, err)
		require.Len(t, dps, 1)
		assert.Equal(t, float64(99.0), dps[0].Value, "should prefer column named 'value'")
	})

	t.Run("first_numeric_fallback", func(t *testing.T) {
		rs := &ljson.ResultSet{
			Schema: &ljson.Schema{
				Columns: []ljson.ColumnMetadata{
					{Name: "cpu_usage", DataType: "float64"}, // first → representative
					{Name: "mem_usage", DataType: "float64"},
				},
			},
			Rows: [][]any{{float64(42.0), float64(80.0)}},
		}
		ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

		dps, err := ar.FetchData()
		require.NoError(t, err)
		require.Len(t, dps, 1)
		assert.Equal(t, float64(42.0), dps[0].Value, "should fall back to first numeric column")
	})
}

// TestFetchData_BoolColumnsInFields verifies that bool (and other non-numeric, non-string)
// columns are placed in Fields, not Values or Labels.
func TestFetchData_BoolColumnsInFields(t *testing.T) {
	rs := &ljson.ResultSet{
		Schema: &ljson.Schema{
			Columns: []ljson.ColumnMetadata{
				{Name: "host", DataType: "string"},
				{Name: "cpu_usage", DataType: "float64"},
				{Name: "is_primary", DataType: "bool"},
			},
		},
		Rows: [][]any{{"web1", float64(50.0), true}},
	}
	ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

	dps, err := ar.FetchData()
	require.NoError(t, err)
	require.Len(t, dps, 1)

	dp := dps[0]
	assert.Equal(t, float64(50.0), dp.Values["cpu_usage"])
	assert.Equal(t, true, dp.Fields["is_primary"])
	_, inLabels := dp.Labels["is_primary"]
	assert.False(t, inLabels, "bool column must not appear in Labels")
	_, inValues := dp.Values["is_primary"]
	assert.False(t, inValues, "bool column must not appear in Values")
}

// TestFetchData_TimeSeriesExpanded verifies that time_series columns expand to one DataPoint
// per time point, and that same-row scalar columns are merged into every time point's Values.
func TestFetchData_TimeSeriesExpanded(t *testing.T) {
	ts := &ljson.TimeSeries{
		Values:    []float64{10.0, 20.0, 30.0},
		TimeRange: ljson.TimeRange{Start: 100, End: 300},
		Interval:  100,
	}
	rs := &ljson.ResultSet{
		Schema: &ljson.Schema{
			Columns: []ljson.ColumnMetadata{
				{Name: "host", DataType: "string"},
				{Name: "cpu", DataType: "time_series"},
				{Name: "threshold", DataType: "float64"}, // scalar alongside time_series
			},
		},
		Rows: [][]any{{"db1", ts, float64(25.0)}},
	}
	ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

	dps, err := ar.FetchData()
	require.NoError(t, err)
	// Three time points → three DataPoints
	assert.Len(t, dps, 3)

	for i, dp := range dps {
		assert.Equal(t, map[string]string{"host": "db1"}, dp.Labels)
		assert.Equal(t, ts.Values[i], dp.Value, "representative value is the time-series point value")
		assert.Equal(t, ts.Values[i], dp.Values["cpu"])
		assert.Equal(t, float64(25.0), dp.Values["threshold"],
			"scalar column must be merged into each time-series DataPoint")
		assert.Equal(t, ts.TimeRange.Start+int64(i)*ts.Interval, dp.Timestamp)
	}
}

// TestFetchData_StringOnlyRowEmitsNoDataPoint ensures that a row with only string columns
// does not produce a DataPoint (no scalar, no time_series).
func TestFetchData_StringOnlyRowEmitsNoDataPoint(t *testing.T) {
	rs := &ljson.ResultSet{
		Schema: &ljson.Schema{
			Columns: []ljson.ColumnMetadata{
				{Name: "host", DataType: "string"},
			},
		},
		Rows: [][]any{{"server1"}},
	}
	ar := newTestRule(t, testDataJSON, &stubQuerier{result: map[string]*ljson.ResultSet{"A": rs}})

	dps, err := ar.FetchData()
	require.NoError(t, err)
	assert.Empty(t, dps)
}

// TestFetchData_FreshTimeRange verifies the staleness fix: FetchData must re-parse the rule's
// Data JSON on every call so that relative time expressions like "now-5m" stay current.
// We use a fixed numeric range and assert the querier receives exactly those timestamps.
func TestFetchData_FreshTimeRange(t *testing.T) {
	var capturedRange model.TimeRange
	stub := &captureQuerier{
		onQuery: func(_ []*model.Query, tr model.TimeRange) (map[string]*ljson.ResultSet, error) {
			capturedRange = tr
			return map[string]*ljson.ResultSet{}, nil
		},
	}
	dataJSON := `{"range":{"start":5000,"end":9000},"queries":[{"refId":"A","datasource":{"uid":"ds1"},"params":{}}]}`
	ar := newTestRule(t, dataJSON, stub)

	_, err := ar.FetchData()
	require.NoError(t, err)
	assert.Equal(t, int64(5000), capturedRange.Start)
	assert.Equal(t, int64(9000), capturedRange.End)
}

// ----------------------------------------------------------------------------
// Eval tests
// ----------------------------------------------------------------------------

// TestEval_SeveritySetOnMatch checks that matched DataPoints receive the correct Severity
// and that unmatched ones are excluded from the result.
func TestEval_SeveritySetOnMatch(t *testing.T) {
	// condition: value > 50, severity = AlertSeverity (1)
	rule := &model.AlertRule{
		UID:        "test-rule",
		Version:    1,
		Data:       datatypes.JSON(testDataJSON),
		Conditions: datatypes.JSON(`[{"severity":1,"expr":"value > 50"}]`),
	}
	ar := NewAlertRule(rule, &Deps{})
	ar.querier = &stubQuerier{}
	require.NoError(t, ar.Perpare())

	dps := []*model.DataPoint{
		{Value: 80.0, Values: map[string]float64{"value": 80.0}},  // matches
		{Value: 30.0, Values: map[string]float64{"value": 30.0}},  // does not match
		{Value: 100.0, Values: map[string]float64{"value": 100.0}}, // matches
	}

	matched, err := ar.Eval(context.Background(), dps)
	require.NoError(t, err)
	require.Len(t, matched, 2)
	for _, dp := range matched {
		assert.Equal(t, model.Severity(model.AlertSeverity), dp.Severity)
	}
}

// TestEval_FirstMatchingConditionWins ensures the eval loop breaks after the first
// matching condition so a DataPoint is never appended more than once.
func TestEval_FirstMatchingConditionWins(t *testing.T) {
	// Two conditions both match value=100; DataPoint should appear exactly once.
	rule := &model.AlertRule{
		UID:     "test-rule",
		Version: 1,
		Data:    datatypes.JSON(testDataJSON),
		// severity 1 (AlertSeverity) and severity 2 (WarningSeverity) both fire for value=100
		Conditions: datatypes.JSON(`[{"severity":1,"expr":"value > 50"},{"severity":2,"expr":"value > 80"}]`),
	}
	ar := NewAlertRule(rule, &Deps{})
	ar.querier = &stubQuerier{}
	require.NoError(t, ar.Perpare())

	dps := []*model.DataPoint{
		{Value: 100.0, Values: map[string]float64{"value": 100.0}},
	}

	matched, err := ar.Eval(context.Background(), dps)
	require.NoError(t, err)
	// The DataPoint must appear exactly once, with the first matching severity.
	require.Len(t, matched, 1)
	assert.Equal(t, model.Severity(model.AlertSeverity), matched[0].Severity)
}

// ----------------------------------------------------------------------------
// DataPoint.ToMap tests
// ----------------------------------------------------------------------------

// TestDataPoint_ToMap_Priority verifies the key priority: Values > Fields > Labels.
// A key present in all three maps must resolve to the Values value.
func TestDataPoint_ToMap_Priority(t *testing.T) {
	dp := &model.DataPoint{
		RefID: "A",
		Labels: map[string]string{
			"host":  "srv1",
			"clash": "from-labels",
		},
		Fields: map[string]any{
			"extra": "field-value",
			"clash": "from-fields", // should be overridden by Values
		},
		Values: map[string]float64{
			"cpu":   75.0,
			"clash": 999.0, // highest priority
		},
		Timestamp: 12345,
		Value:     75.0,
	}

	env := dp.ToMap()

	// fixed keys always present
	assert.Equal(t, "A", env["refId"])
	assert.Equal(t, float64(75.0), env["value"])
	assert.Equal(t, int64(12345), env["timestamp"])

	// each map contributes its own keys
	assert.Equal(t, "srv1", env["host"])
	assert.Equal(t, "field-value", env["extra"])
	assert.Equal(t, float64(75.0), env["cpu"])

	// clash resolution: Values wins over Fields wins over Labels
	assert.Equal(t, float64(999.0), env["clash"], "Values must override Fields and Labels")
}
