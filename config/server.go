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

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/caarlos0/env/v7"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/ltoml"

	aiconfig "github.com/lindb/linsight/ai/config"
	"github.com/lindb/linsight/constant"
)

// for testing
var (
	loadConfigFn = ltoml.LoadConfig
	envParseFn   = env.Parse
)

var (
	globalServerCfg atomic.Value
)

// HTTP represents an HTTP level configuration of server.
type HTTP struct {
	Port         uint16         `env:"PORT" toml:"port"`
	IdleTimeout  ltoml.Duration `env:"IDLE_TIMEOUT" toml:"idle-timeout"`
	WriteTimeout ltoml.Duration `env:"WRITE_TIMEOUT" toml:"write-timeout"`
	ReadTimeout  ltoml.Duration `env:"READ_TIMEOUT" toml:"read-timeout"`
}

func (h *HTTP) TOML() string {
	return fmt.Sprintf(`
## Port the HTTP server listens on.
## Default: %d
## Env: LINSIGHT_HTTP_PORT
port = %d
## Maximum duration to keep idle connections alive.
## Default: %s
## Env: LINSIGHT_HTTP_IDLE_TIMEOUT
idle-timeout = "%s"
## Maximum duration before timing out writes of the response.
## Default: %s
## Env: LINSIGHT_HTTP_WRITE_TIMEOUT
write-timeout = "%s"
## Maximum duration for reading the entire request, including the body.
## Default: %s
## Env: LINSIGHT_HTTP_READ_TIMEOUT
read-timeout = "%s"`,
		h.Port, h.Port,
		h.IdleTimeout.Duration().String(), h.IdleTimeout.Duration().String(),
		h.WriteTimeout.Duration().String(), h.WriteTimeout.Duration().String(),
		h.ReadTimeout.Duration().String(), h.ReadTimeout.Duration().String())
}

type Server struct {
	Migration    bool              `env:"LINSIGHT_MIGRATION" toml:"migration"`
	Database     *Database         `envPrefix:"LINSIGHT_DATABASE_" toml:"database"`
	HTTP         *HTTP             `envPrefix:"LINSIGHT_HTTP_" toml:"http"`
	Cookie       *Cookie           `envPrefix:"LINSIGHT_COOKIE_" toml:"cookie"`
	Provisioning string            `env:"LINSIGHT_PROVISIONING" toml:"provisioning"`
	Logger       *logger.Setting   `envPrefix:"LINSIGHT_LOGGING_" toml:"logging"`
	AI           *aiconfig.Config `envPrefix:"LINSIGHT_AI_" toml:"ai"`
	Alerting     *Alerting        `envPrefix:"LINSIGHT_ALERTING_" toml:"alerting"`
}

// TOML returns the server configuration as a commented TOML string.
func (s *Server) TOML() string {
	aiSection := ""
	if s.AI != nil {
		aiSection = fmt.Sprintf("\n\n## AI agent configuration.\n[ai]%s", s.AI.TOML())
	}
	alertingSection := ""
	if s.Alerting != nil {
		alertingSection = fmt.Sprintf("\n\n## Alerting background services configuration.\n[alerting]%s", s.Alerting.TOML())
	}
	return fmt.Sprintf(`## Run database migrations automatically on every startup.
## Default: %v
## Env: LINSIGHT_MIGRATION
migration = %v

## Directory that LinSight scans for provisioning files (dashboards, etc.).
## Default: %s
## Env: LINSIGHT_PROVISIONING
provisioning = "%s"

## Database configuration.
[database]%s

## HTTP server configuration.
[http]%s

## Session cookie configuration.
[cookie]%s
%s%s%s`,
		s.Migration, s.Migration,
		s.Provisioning, s.Provisioning,
		s.Database.TOML(),
		s.HTTP.TOML(),
		s.Cookie.TOML(),
		s.Logger.TOML("LINSIGHT"),
		aiSection,
		alertingSection,
	)
}

func NewDefaultServer() *Server {
	return &Server{
		Migration: true,
		Database: &Database{
			Type:  "sqlite",
			DSN:   "linsight.db",
			Debug: true,
		},
		HTTP: &HTTP{
			Port:         8080,
			IdleTimeout:  ltoml.Duration(time.Minute),
			WriteTimeout: ltoml.Duration(time.Second * 30),
			ReadTimeout:  ltoml.Duration(time.Second * 30),
		},
		Provisioning: filepath.Join(".", "data", "provisioning"),
		Cookie: &Cookie{
			Name:   constant.LinSightCookie,
			MaxAge: ltoml.Duration(time.Hour * 24 * 30),
		},
		Logger: logger.NewDefaultSetting(),
		AI:       aiconfig.NewDefaultConfig(),
		Alerting: NewDefaultAlerting(),
	}
}

// NewDefaultServerTOML returns the default server configuration as a commented TOML string,
// suitable for writing to an initial configuration file (linsight init-config).
func NewDefaultServerTOML() string {
	return NewDefaultServer().TOML()
}

func LoadAndSetServerConfig(cfgName, defaultPath string, serverCfg *Server) error {
	if err := loadConfigFn(cfgName, defaultPath, serverCfg); err != nil {
		return fmt.Errorf("decode server config file error: %s", err)
	}
	if err := envParseFn(serverCfg); err != nil {
		return fmt.Errorf("read server env error: %s", err)
	}
	globalServerCfg.Store(serverCfg)
	return nil
}
