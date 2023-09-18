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

package http

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/lindb/common/pkg/http/middleware"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/config"
)

// for testing.
var (
	listenFn = net.Listen
)

// Server represents http server based on gin.
type Server struct {
	engine *gin.Engine
	addr   string
	server http.Server
}

// NewServer creates a http server instance.
func NewServer(cfg *config.HTTP) *Server {
	mode := os.Getenv(gin.EnvGinMode)
	if mode == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	server := &Server{
		engine: gin.New(),
		addr:   fmt.Sprintf(":%d", cfg.Port),
		server: http.Server{
			// use extra timeout for ingestion and query timeout
			// if write timeout will return ERR_EMPTY_RESPONSE, chrome will does auto retry.
			// https://www.bennadel.com/blog/3257-google-chrome-will-automatically-retry-requests-on-certain-error-responses.htm
			// https://mariocarrion.com/2021/09/17/golang-software-architecture-resilience-http-servers.html
			WriteTimeout: cfg.WriteTimeout.Duration(),
			ReadTimeout:  cfg.ReadTimeout.Duration(),
		},
	}
	server.initialize()
	return server
}

// initialize initializes gin engine.
// 1. add common middleware
// 2. add static resource handle
func (s *Server) initialize() {
	// Using middlewares on group.
	s.engine.Use(middleware.Recovery())
	// use AccessLog to log panic error with zap
	s.engine.Use(middleware.AccessLog(logger.GetLogger(logger.AccessLogModule, "HTTP")))
	s.engine.Use(cors.Default())
	// handle static resource
	handleStatic(s.engine)
}

// Run runs http server.
func (s *Server) Run() error {
	s.server.Handler = s.engine
	listen, err := listenFn("tcp", s.addr)
	if err != nil {
		return err
	}
	return s.server.Serve(listen)
}

// GetEngine returns gin engine.
func (s *Server) GetEngine() *gin.Engine {
	return s.engine
}
