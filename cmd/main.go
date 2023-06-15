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

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lindb/common/pkg/fileutil"

	"github.com/lindb/linsight/config"
	"github.com/lindb/linsight/http"
	"github.com/lindb/linsight/http/deps"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

const (
	serverCfgName        = "linsight.toml"
	serverLogFileName    = "linsight.log"
	defaultServerCfgFile = "./" + serverCfgName
)

var (
	cfgFile = ""
)

// RootCmd command of cobra
var RootCmd = &cobra.Command{
	Use:   "sight",
	Short: "sight is the main command, used to control LinSight",
	Long:  "logo and desc",
}

func init() {
	RootCmd.AddCommand(
		newServerCmd(),
		newMigratorCmd(),
	)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newServerCmd() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run as a server",
	}
	serverCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		fmt.Sprintf("linsight config file path, default is %s", defaultServerCfgFile))
	serverCmd.AddCommand(
		&cobra.Command{
			Use:   "run",
			Short: "starts the server",
			RunE:  runServer,
		},
	)
	return serverCmd
}

func runServer(_ *cobra.Command, _ []string) error {
	ctx := newCtxWithSignals()
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}

	return run(ctx, func() {
		db, err := dbpkg.NewDB(cfg.Database)
		if err != nil {
			panic(err)
		}
		go func() {
			apiServer := http.NewServer(cfg.HTTP)
			engine := apiServer.GetEngine()
			router := http.NewRouter(engine, buildDeps(db, cfg))
			router.RegisterRouters()
			if err := apiServer.Run(); err != nil {
				panic(err)
			}
		}()
	})
}

func buildDeps(db dbpkg.DB, cfg *config.Server) *deps.API {
	authorizeSrv := service.NewAuthorizeService(db)
	// initialize access roles and policies
	if err := authorizeSrv.Initialize(); err != nil {
		panic(err)
	}
	orgSrv := service.NewOrgService(db)
	userSrv := service.NewUserService(db, orgSrv)
	starSrv := service.NewStarService(db)
	return &deps.API{
		Config:          cfg,
		OrgSrv:          orgSrv,
		UserSrv:         userSrv,
		TeamSrv:         service.NewTeamService(db),
		NavSrv:          service.NewNavService(db),
		AuthorizeSrv:    authorizeSrv,
		AuthenticateSrv: service.NewAuthenticateService(userSrv, db),
		DatasourceSrv:   service.NewDatasourceService(db),
		DashboardSrv:    service.NewDashboardService(starSrv, db),
		ChartSrv:        service.NewChartService(db),

		DatasourceMgr: datasource.NewDatasourceManager(),
	}
}

func loadConfig() (*config.Server, error) {
	var cfg *config.Server
	if fileutil.Exist(cfgFile) || fileutil.Exist(defaultServerCfgFile) {
		cfg = &config.Server{}
		if err := config.LoadAndSetServerConfig(cfgFile, defaultServerCfgFile, cfg); err != nil {
			return nil, err
		}
	} else {
		// if not set config file, using default config
		cfg = config.NewDefaultServer()
	}
	return cfg, nil
}
