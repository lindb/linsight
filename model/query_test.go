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

package model

import (
	"encoding/json"
	"testing"

	"github.com/lindb/common/pkg/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedNow is a constant used to make relative-time assertions deterministic.
const fixedNow int64 = 1_700_000_000_000 // arbitrary millisecond timestamp

func TestTimeRange_UnmarshalJSON_Number(t *testing.T) {
	data := []byte(`{"start":1000000,"end":2000000}`)
	var tr TimeRange
	require.NoError(t, json.Unmarshal(data, &tr))
	assert.Equal(t, int64(1000000), tr.Start)
	assert.Equal(t, int64(2000000), tr.End)
}

func TestTimeRange_UnmarshalJSON_StringNow(t *testing.T) {
	defer func() { nowFn = timeutil.Now }()
	nowFn = func() int64 { return fixedNow }

	data := []byte(`{"start":"now","end":"now"}`)
	var tr TimeRange
	require.NoError(t, json.Unmarshal(data, &tr))
	assert.Equal(t, fixedNow, tr.Start)
	assert.Equal(t, fixedNow, tr.End)
}

func TestTimeRange_UnmarshalJSON_RelativeTime(t *testing.T) {
	defer func() { nowFn = timeutil.Now }()
	nowFn = func() int64 { return fixedNow }

	cases := []struct {
		name      string
		input     string
		wantStart int64
		wantEnd   int64
	}{
		{
			name:      "now-5m / now",
			input:     `{"start":"now-5m","end":"now"}`,
			wantStart: fixedNow - 5*timeutil.OneMinute,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-1h / now",
			input:     `{"start":"now-1h","end":"now"}`,
			wantStart: fixedNow - timeutil.OneHour,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-1d / now",
			input:     `{"start":"now-1d","end":"now"}`,
			wantStart: fixedNow - timeutil.OneDay,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-1w / now",
			input:     `{"start":"now-1w","end":"now"}`,
			wantStart: fixedNow - timeutil.OneWeek,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-1M / now",
			input:     `{"start":"now-1M","end":"now"}`,
			wantStart: fixedNow - timeutil.OneMonth,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-1y / now",
			input:     `{"start":"now-1y","end":"now"}`,
			wantStart: fixedNow - timeutil.OneYear,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-30s / now",
			input:     `{"start":"now-30s","end":"now"}`,
			wantStart: fixedNow - 30*timeutil.OneSecond,
			wantEnd:   fixedNow,
		},
		{
			name:      "now-15m string as start, numeric as end",
			input:     `{"start":"now-15m","end":2000000}`,
			wantStart: fixedNow - 15*timeutil.OneMinute,
			wantEnd:   2000000,
		},
		{
			// n=0 is valid and should equal "now".
			name:      "now-0m equals now",
			input:     `{"start":"now-0m","end":"now"}`,
			wantStart: fixedNow,
			wantEnd:   fixedNow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tr TimeRange
			require.NoError(t, json.Unmarshal([]byte(tc.input), &tr))
			assert.Equal(t, tc.wantStart, tr.Start)
			assert.Equal(t, tc.wantEnd, tr.End)
		})
	}
}

func TestTimeRange_UnmarshalJSON_StringInteger(t *testing.T) {
	defer func() { nowFn = timeutil.Now }()
	nowFn = func() int64 { return fixedNow }

	// A plain integer string should be treated as a millisecond timestamp.
	data := []byte(`{"start":"1000000","end":"2000000"}`)
	var tr TimeRange
	require.NoError(t, json.Unmarshal(data, &tr))
	assert.Equal(t, int64(1000000), tr.Start)
	assert.Equal(t, int64(2000000), tr.End)
}

func TestTimeRange_UnmarshalJSON_MissingFields(t *testing.T) {
	// Absent fields should default to zero without error.
	data := []byte(`{}`)
	var tr TimeRange
	require.NoError(t, json.Unmarshal(data, &tr))
	assert.Equal(t, int64(0), tr.Start)
	assert.Equal(t, int64(0), tr.End)

	// Explicit JSON null should also default to zero.
	data = []byte(`{"start":null,"end":null}`)
	require.NoError(t, json.Unmarshal(data, &tr))
	assert.Equal(t, int64(0), tr.Start)
	assert.Equal(t, int64(0), tr.End)
}

func TestTimeRange_UnmarshalJSON_InvalidExpressions(t *testing.T) {
	defer func() { nowFn = timeutil.Now }()
	nowFn = func() int64 { return fixedNow }

	cases := []struct {
		name  string
		input string
	}{
		{"now- with no value", `{"start":"now-","end":"now"}`},
		{"unknown unit", `{"start":"now-5x","end":"now"}`},
		{"negative number", `{"start":"now--1h","end":"now"}`},
		{"non-numeric prefix", `{"start":"now-abch","end":"now"}`},
		{"completely invalid string", `{"start":"yesterday","end":"now"}`},
		// Boolean values must be rejected, not silently coerced to 0.
		{"boolean false as start", `{"start":false,"end":"now"}`},
		{"boolean true as start", `{"start":true,"end":"now"}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tr TimeRange
			assert.Error(t, json.Unmarshal([]byte(tc.input), &tr))
		})
	}
}

func TestParseRelativeTime(t *testing.T) {
	defer func() { nowFn = timeutil.Now }()
	nowFn = func() int64 { return fixedNow }

	v, err := parseRelativeTime("now")
	require.NoError(t, err)
	assert.Equal(t, fixedNow, v)

	v, err = parseRelativeTime("now-2h")
	require.NoError(t, err)
	assert.Equal(t, fixedNow-2*timeutil.OneHour, v)

	_, err = parseRelativeTime("invalid")
	assert.Error(t, err)
}
