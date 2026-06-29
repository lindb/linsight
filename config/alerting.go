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

package config

import "fmt"

// Alerting holds configuration for alerting background services.
// When Enabled is false, the rule evaluation scheduler, repeat-notification
// ticker, and alert group cleanup goroutine are not started at server boot.
type Alerting struct {
	// Enabled gates all alerting background services. Default: true.
	Enabled bool `env:"ENABLED" toml:"enabled"`
}

// TOML returns the alerting configuration as a commented TOML string fragment.
func (a *Alerting) TOML() string {
	return fmt.Sprintf(`
## Enable alerting background services (rule evaluation scheduler, repeat-notification ticker, group cleanup).
## Default: %v
## Env: LINSIGHT_ALERTING_ENABLED
enabled = %v`, a.Enabled, a.Enabled)
}

// NewDefaultAlerting returns an Alerting config with background services enabled.
// Defaults to true to preserve existing behaviour for users who upgrade without
// adding an [alerting] section to their configuration file.
func NewDefaultAlerting() *Alerting {
	return &Alerting{Enabled: true}
}
