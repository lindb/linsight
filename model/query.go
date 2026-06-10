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
	"fmt"
	"strconv"
	"strings"

	"github.com/lindb/common/pkg/timeutil"
	"github.com/tidwall/gjson"
)

// for testing
var nowFn = timeutil.Now

// TimeRange represents data query time range.
// Both Start and End accept either a millisecond Unix timestamp (int64)
// or a relative time string such as "now", "now-5m", "now-1h".
type TimeRange struct {
	Start int64 `json:"start"`
	End   int64 `json:"end"`
}

// UnmarshalJSON implements json.Unmarshaler so that Start/End can be decoded
// from either a JSON number (millisecond timestamp) or a relative-time string.
func (t *TimeRange) UnmarshalJSON(data []byte) error {
	raw := string(data)

	startResult := gjson.Get(raw, "start")
	endResult := gjson.Get(raw, "end")

	start, err := parseTimeField(startResult)
	if err != nil {
		return fmt.Errorf("invalid TimeRange.Start: %w", err)
	}
	end, err := parseTimeField(endResult)
	if err != nil {
		return fmt.Errorf("invalid TimeRange.End: %w", err)
	}

	t.Start = start
	t.End = end
	return nil
}

// parseTimeField converts a gjson.Result to a millisecond timestamp.
// It handles JSON number types directly and parses string values as
// relative time expressions (e.g. "now", "now-5m").
func parseTimeField(r gjson.Result) (int64, error) {
	switch r.Type {
	case gjson.Number:
		return r.Int(), nil
	case gjson.String:
		return parseRelativeTime(r.String())
	case gjson.Null: // field absent or explicitly null → default to zero
		return 0, nil
	default:
		// Covers gjson.False, gjson.True, gjson.JSON, etc.
		return 0, fmt.Errorf("unexpected type %s", r.Type)
	}
}

// parseRelativeTime parses a relative time expression and returns the
// corresponding millisecond Unix timestamp.
//
// Supported formats:
//   - "now"           → current time
//   - "now-{N}{unit}" → current time minus N units
//   - plain integer   → parsed as millisecond timestamp directly
//
// Units: s=seconds, m=minutes, h=hours, d=days, w=weeks, M=months, y=years
func parseRelativeTime(s string) (int64, error) {
	if s == "now" {
		return nowFn(), nil
	}

	if strings.HasPrefix(s, "now-") {
		expr := s[4:] // strip "now-"
		if len(expr) < 2 {
			return 0, fmt.Errorf("invalid relative time expression: %q", s)
		}

		// Last character is the unit; everything before is the numeric value.
		unit := expr[len(expr)-1]
		numStr := expr[:len(expr)-1]
		n, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid relative time expression: %q", s)
		}

		var duration int64
		switch unit {
		case 's':
			duration = n * timeutil.OneSecond
		case 'm':
			duration = n * timeutil.OneMinute
		case 'h':
			duration = n * timeutil.OneHour
		case 'd':
			duration = n * timeutil.OneDay
		case 'w':
			duration = n * timeutil.OneWeek
		case 'M':
			duration = n * timeutil.OneMonth
		case 'y':
			duration = n * timeutil.OneYear
		default:
			return 0, fmt.Errorf("unknown time unit %q in expression: %q", string(unit), s)
		}
		return nowFn() - duration, nil
	}

	// Fallback: treat the string as a plain integer millisecond timestamp.
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cannot parse time value: %q", s)
	}
	return v, nil
}

type QueryRequest struct {
	Range   TimeRange `json:"range"`
	Queries []*Query  `json:"queries"`
}

type Query struct {
	Datasource TargetDatasource `json:"datasource" binding:"required"`
	Params     json.RawMessage  `json:"params"`
	RefID      string           `json:"refId"`
}

type TargetDatasource struct {
	UID string `json:"uid" binding:"required"`
}

type QueryResponse struct {
	Results map[string]any `json:"results"`
}
