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
	"sync/atomic"
	"time"

	"github.com/caarlos0/env/v7"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/ltoml"

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

type Server struct {
	Database *Database       `envPrefix:"LINSIGHT_DATABASE_" toml:"database"`
	HTTP     *HTTP           `envPrefix:"LINSIGHT_HTTP_" toml:"http"`
	Cookie   *Cookie         `envPrefix:"LINSIGHT_COOKIE_" toml:"cookie"`
	Logger   *logger.Setting `envPrefix:"LINSIGHT_LOGGER_" toml:"logger"`
}

func NewDefaultServer() *Server {
	return &Server{
		Database: &Database{
			Type:  "sqlite",
			DSN:   "linsight.db",
			Debug: true,
		},
		HTTP: &HTTP{
			Port:         8080,
			WriteTimeout: ltoml.Duration(time.Second * 30),
			ReadTimeout:  ltoml.Duration(time.Second * 30),
		},
		Cookie: &Cookie{
			Name:   constant.LinSightCookie,
			MaxAge: ltoml.Duration(time.Hour * 24 * 30),
		},
		Logger: logger.NewDefaultSetting(),
	}
}

func LoadAndSetServerConfig(cfgName, defaultPath string, serverCfg *Server) error {
	if err := loadConfigFn(cfgName, defaultPath, &serverCfg); err != nil {
		return fmt.Errorf("decode server config file error: %s", err)
	}
	if err := envParseFn(serverCfg); err != nil {
		return fmt.Errorf("read server env error: %s", err)
	}
	globalServerCfg.Store(serverCfg)
	return nil
}
