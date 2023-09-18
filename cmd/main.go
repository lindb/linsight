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
	"time"

	"github.com/spf13/cobra"

	"github.com/lindb/common/pkg/fileutil"
	"github.com/lindb/common/pkg/logger"

	_ "github.com/lindb/linsight/alerting"
	"github.com/lindb/linsight/alerting/schedule"
	"github.com/lindb/linsight/config"
	"github.com/lindb/linsight/http"
	"github.com/lindb/linsight/http/deps"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/plugin/datasource"
	provisioningdeps "github.com/lindb/linsight/provisioning/deps"
	provisionservice "github.com/lindb/linsight/provisioning/service"
	"github.com/lindb/linsight/service"
)

const (
	serverCfgName        = "linsight.toml"
	serverLogFileName    = "linsight.log"
	defaultServerCfgFile = "./" + serverCfgName
)

var (
	log     = logger.GetLogger("CMD", "Main")
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

	if cfg.Migration {
		// if auto migration enabled, need do data migration
		if err := runMigration(nil, nil); err != nil {
			panic(err)
		}
	}

	return run(ctx, func() {
		db, err := dbpkg.NewDB(cfg.Database)
		if err != nil {
			panic(err)
		}
		go func() {
			apiServer := http.NewServer(cfg.HTTP)
			engine := apiServer.GetEngine()
			apiDeps := buildDeps(db, cfg)
			router := http.NewRouter(engine, apiDeps)
			router.RegisterRouters()

			// TODO:
			sche := schedule.NewSchedule(
				ctx,
				1,
				&schedule.Deps{
					Opts: &schedule.Options{
						Interval:        time.Second * 10,
						MinEvalInterval: time.Second * 10,
					},
					DatasourceSrv:   apiDeps.DatasourceSrv,
					AlertRuleSrv:    apiDeps.AlertRuleSrv,
					NotificationSrv: apiDeps.NotificationSrv,
					DatasourceMgr:   apiDeps.DatasourceMgr,
				},
			)
			sche.Run()

			provisionSrv := provisionservice.NewProvisionService(&provisioningdeps.ProvisioningDeps{
				BaseDir:      cfg.Provisioning,
				OrgSrv:       apiDeps.OrgSrv,
				DashboardSrv: apiDeps.DashboardSrv,
			})
			defer func() {
				if err := provisionSrv.Shutdown(); err != nil {
					log.Error("shutdown provisioning service failure", logger.Error(err))
				}
				if err := db.Close(); err != nil {
					log.Error("close db failure", logger.Error(err))
				}
			}()
			if err := provisionSrv.Run(); err != nil {
				panic(err)
			}

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
	cmpSrv := service.NewComponentService(db, orgSrv, authorizeSrv)
	// initialize supported component
	if err := cmpSrv.Initialize(); err != nil {
		panic(err)
	}
	integrationSrv := service.NewIntegrationService(db)
	// initialize supported integration
	if err := integrationSrv.Initialize(); err != nil {
		panic(err)
	}
	userSrv := service.NewUserService(db, orgSrv)
	starSrv := service.NewStarService(db)
	tagSrv := service.NewTagService(db)
	return &deps.API{
		Config:          cfg,
		OrgSrv:          orgSrv,
		UserSrv:         userSrv,
		TeamSrv:         service.NewTeamService(db, userSrv),
		CmpSrv:          cmpSrv,
		IntegrationSrv:  integrationSrv,
		AuthorizeSrv:    authorizeSrv,
		AuthenticateSrv: service.NewAuthenticateService(userSrv, db),
		TagSrv:          tagSrv,
		DatasourceSrv:   service.NewDatasourceService(db),
		DashboardSrv:    service.NewDashboardService(starSrv, tagSrv, db),
		ChartSrv:        service.NewChartService(db),

		NotificationSrv: service.NewNotificationService(db),
		AlertRuleSrv:    service.NewAlertRuleService(db),

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
