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

type Database struct {
	Type  string `env:"TYPE" toml:"type"`
	DSN   string `env:"DSN" toml:"dsn"`
	Debug bool   `env:"DEBUG" toml:"debug"`
}

func (d *Database) TOML() string {
	return fmt.Sprintf(`
## Database driver: sqlite, mysql, postgres.
## Default: %s
## Env: LINSIGHT_DATABASE_TYPE
type = "%s"
## Data source name. For SQLite this is the file path; for MySQL/Postgres use a DSN connection string.
## Default: %s
## Env: LINSIGHT_DATABASE_DSN
dsn = "%s"
## Enable SQL debug logging (prints all queries).
## Default: %v
## Env: LINSIGHT_DATABASE_DEBUG
debug = %v`,
		d.Type, d.Type,
		d.DSN, d.DSN,
		d.Debug, d.Debug)
}
