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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Duration wraps time.Duration so that it serializes as a human-readable string in JSON
// (e.g. "30s", "5m", "4h") while still persisting as int64 nanoseconds in the database.
// Zero duration marshals as JSON null.
type Duration time.Duration

// MarshalJSON encodes the duration as a quoted string ("30s", "5m", …), or null when zero.
// Zero sub-components are stripped: "2h0m0s" → "2h", "1h30m0s" → "1h30m".
func (d Duration) MarshalJSON() ([]byte, error) {
	if d == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(trimDuration(time.Duration(d)))
}

// UnmarshalJSON decodes a quoted duration string or JSON null.
func (d *Duration) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*d = 0
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*d = 0
		return nil
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(dur)
	return nil
}

// trimDuration formats d omitting zero sub-components.
// "2h0m0s" → "2h", "1h30m0s" → "1h30m", "5m0s" → "5m".
// Sub-second durations fall back to time.Duration.String().
func trimDuration(d time.Duration) string {
	if d < time.Second {
		return d.String()
	}
	total := int64(d / time.Second)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60

	var buf strings.Builder
	if h > 0 {
		fmt.Fprintf(&buf, "%dh", h)
	}
	if m > 0 {
		fmt.Fprintf(&buf, "%dm", m)
	}
	if s > 0 {
		fmt.Fprintf(&buf, "%ds", s)
	}
	return buf.String()
}

// Value implements driver.Valuer so GORM stores the duration as int64 nanoseconds.
func (d Duration) Value() (driver.Value, error) {
	return int64(d), nil
}

// Scan implements sql.Scanner so GORM can read the stored int64 back into Duration.
func (d *Duration) Scan(value interface{}) error {
	if value == nil {
		*d = 0
		return nil
	}
	switch v := value.(type) {
	case int64:
		*d = Duration(v)
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Duration", value)
	}
}

// Milliseconds returns the duration in whole milliseconds (delegates to time.Duration).
func (d Duration) Milliseconds() int64 { return time.Duration(d).Milliseconds() }

// Seconds returns the duration as a floating-point number of seconds (delegates to time.Duration).
func (d Duration) Seconds() float64 { return time.Duration(d).Seconds() }
