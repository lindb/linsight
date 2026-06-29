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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/lindb/common/pkg/fileutil"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/ltoml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	linsight "github.com/lindb/linsight"
	_ "github.com/lindb/linsight/alerting"
	"github.com/lindb/linsight/alerting/manager"
	"github.com/lindb/linsight/alerting/schedule"
	alerttemplate "github.com/lindb/linsight/alerting/template"
	aimodule "github.com/lindb/linsight/ai"
	"github.com/lindb/linsight/config"
	"github.com/lindb/linsight/http"
	"github.com/lindb/linsight/http/deps"
	"github.com/lindb/linsight/model"
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

// RootCmd is the root command of the CLI.
var RootCmd = &cobra.Command{
	Use:   "linsight",
	Short: "LinSight - unified observability platform",
	Long:  "LinSight is an open-source observability platform for metrics, logs, and traces.",
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the LinSight server",
	Long:  "Start the LinSight HTTP server and all background services (alerting scheduler, provisioning, etc.).",
	RunE:  runServer,
}

var initConfigCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Generate a default configuration file",
	Long:  fmt.Sprintf("Write a default server configuration to a TOML file (default: %s). Prompts before overwriting an existing file.", defaultServerCfgFile),
	RunE: func(_ *cobra.Command, _ []string) error {
		path := cfgFile
		if path == "" {
			path = defaultServerCfgFile
		}
		if err := checkExistenceOf(path); err != nil {
			if errors.Is(err, errSkipped) {
				return nil
			}
			return err
		}
		return ltoml.WriteConfig(path, config.NewDefaultServerTOML())
	},
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		fmt.Sprintf("config file path (default: %s)", defaultServerCfgFile))
	RootCmd.AddCommand(serverCmd, initConfigCmd, migratorCmd)
}

func main() {
	if err := RootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runServer(_ *cobra.Command, _ []string) error {
	ctx := newCtxWithSignals()
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if cfg.Migration {
		// if auto migration enabled, need do data migration
		if err := runMigration(nil, nil); err != nil {
			return fmt.Errorf("auto migration failed: %w", err)
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
					AlertMgr:        apiDeps.AlertMgr,
				},
			)
			sche.Run()

			// start the repeat-notification ticker so steady-state Firing alerts are re-notified
			go apiDeps.AlertMgr.StartRepeatTicker(ctx)

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
	menuSrv := service.NewMenuService(db, orgSrv, authorizeSrv)
	// initialize supported component
	if err := menuSrv.Initialize(); err != nil {
		panic(err)
	}
	integrationSrv := service.NewIntegrationService(db)
	// initialize supported integration
	if err := integrationSrv.Initialize(); err != nil {
		panic(err)
	}
	cache := service.NewUserCacheService()
	userSrv := service.NewUserService(db, orgSrv, cache)
	starSrv := service.NewStarService(db)
	tagSrv := service.NewTagService(db)

	// new alerting services
	alertEventSrv := service.NewAlertEventService(db)
	alertRouteSrv := service.NewAlertRouteService(db)
	contactPointSrv := service.NewContactPointService(db)
	contactPointStatusSrv := service.NewContactPointStatusService(db) // tracks per-channel delivery health
	alertSilenceSrv := service.NewAlertSilenceService(db)
	alertInhibitRuleSrv := service.NewAlertInhibitRuleService(db)
	notificationTemplateSrv := service.NewNotificationTemplateService(db)
	// Parse built-in notification template examples from the embedded YAML file.
	// Each example's payload is a nested YAML object that is re-encoded as a JSON string
	// so that handlers can pass it directly to template.BuildNotificationDataFromPayload.
	notifTmplExamples := mustLoadNotificationTemplateExamples()

	// Build the rendering engine with a TTL cache to avoid per-dispatch DB queries for org templates.
	renderingEngine := alerttemplate.NewRenderingEngine(
		func(orgID int64) (string, error) {
			templates, err := notificationTemplateSrv.GetAllTemplatesByOrgID(orgID)
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			for _, t := range templates {
				sb.WriteString(t.Template)
				sb.WriteByte('\n')
			}
			return sb.String(), nil
		},
		5*time.Minute,
	)

	// alertmanager
	notificationSrv := service.NewNotificationService(db)
	alertRuleGroupSrv := service.NewAlertRuleGroupService(db)
	mgrDeps := &manager.Deps{
		AlertEventSrv:           alertEventSrv,
		AlertRouteSrv:           alertRouteSrv,
		ContactPointSrv:         contactPointSrv,
		AlertSilenceSrv:         alertSilenceSrv,
		AlertInhibitRuleSrv:     alertInhibitRuleSrv,
		NotificationTemplateSrv: notificationTemplateSrv,
		NotificationSrv:         notificationSrv,
		ContactPointStatusSrv:   contactPointStatusSrv, // persist per-channel delivery outcomes
		RenderingEngine:         renderingEngine,
		ExternalURL:             "", // TODO: configure public URL in server config
	}
	alertMgr := manager.NewAlertManager(mgrDeps)

	// Bootstrap the AI module (no-op when cfg.AI is nil or Enabled is false).
	aimodule.Bootstrap(cfg, db)

	return &deps.API{
		Config:              cfg,
		OrgSrv:              orgSrv,
		UserSrv:             userSrv,
		UserCacheSrv:        cache,
		TeamSrv:             service.NewTeamService(db, userSrv),
		MenuSrv:             menuSrv,
		IntegrationSrv:      integrationSrv,
		AuthorizeSrv:        authorizeSrv,
		AuthenticateSrv:     service.NewAuthenticateService(userSrv, db),
		TagSrv:              tagSrv,
		DatasourceSrv:       service.NewDatasourceService(db),
		DashboardSrv:        service.NewDashboardService(starSrv, tagSrv, db),
		DashboardVersionSrv: service.NewDashboardVersionService(db),
		ChartSrv:            service.NewChartService(starSrv, db),

		NotificationSrv:   notificationSrv,
		AlertRuleGroupSrv: alertRuleGroupSrv,
		AlertRuleSrv:      service.NewAlertRuleService(db, alertRuleGroupSrv),

		AlertEventSrv:                alertEventSrv,
		AlertRouteSrv:                alertRouteSrv,
		ContactPointSrv:              contactPointSrv,
		ContactPointStatusSrv:        contactPointStatusSrv,
		AlertSilenceSrv:              alertSilenceSrv,
		AlertInhibitRuleSrv:          alertInhibitRuleSrv,
		NotificationTemplateSrv:      notificationTemplateSrv,
		NotificationTemplateExamples: notifTmplExamples,

		AlertMgr:        alertMgr,
		RenderingEngine: renderingEngine,

		DatasourceMgr: datasource.NewDatasourceManager(),

		// DryRunner reuses the same manager.Deps so it can access silences/inhibits/routes/contact-points.
		DryRunner: schedule.NewDryRunner(&schedule.Deps{
			Opts: &schedule.Options{
				Interval:        time.Second * 10,
				MinEvalInterval: time.Second * 10,
			},
			DatasourceSrv: service.NewDatasourceService(db),
			DatasourceMgr: datasource.NewDatasourceManager(),
			ManagerDeps:   mgrDeps,
		}),
	}
}

// mustLoadNotificationTemplateExamples parses the built-in YAML file embedded in the binary
// and returns a slice of NotificationTemplateExample with Payload encoded as a JSON string.
// Panics on parse error so that misconfigured binaries fail fast at startup.
func mustLoadNotificationTemplateExamples() []model.NotificationTemplateExample {
	// rawEntry is the intermediate YAML representation where payload is a nested map
	// so that yaml.v3 can decode the nested object, which is then re-encoded as JSON.
	type rawEntry struct {
		Name      string                 `yaml:"name"`
		IsDefault bool                   `yaml:"isDefault"`
		Payload   map[string]interface{} `yaml:"payload"`
	}

	var raw []rawEntry
	if err := yaml.Unmarshal([]byte(linsight.NotificationTemplateExamplesYAML), &raw); err != nil {
		panic("failed to parse notification_template_examples.yaml: " + err.Error())
	}

	examples := make([]model.NotificationTemplateExample, 0, len(raw))
	for _, r := range raw {
		payloadJSON, err := json.Marshal(r.Payload)
		if err != nil {
			panic("failed to encode example payload as JSON (name=" + r.Name + "): " + err.Error())
		}
		examples = append(examples, model.NotificationTemplateExample{
			Name:      r.Name,
			IsDefault: r.IsDefault,
			Payload:   string(payloadJSON),
		})
	}
	return examples
}

func loadConfig() (*config.Server, error) {
	// reject an explicit --config path that does not exist
	if cfgFile != "" && !fileutil.Exist(cfgFile) {
		return nil, fmt.Errorf("config file not found: %s", cfgFile)
	}

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
