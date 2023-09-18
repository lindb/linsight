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

	"github.com/spf13/cobra"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/accesscontrol"
	"github.com/lindb/linsight/constant"
	"github.com/lindb/linsight/model"
	dbpkg "github.com/lindb/linsight/pkg/db"
	"github.com/lindb/linsight/pkg/util"
	"github.com/lindb/linsight/pkg/uuid"
)

func newMigratorCmd() *cobra.Command {
	migratorCmd := &cobra.Command{
		Use:   "migrator",
		Short: "Run as a server",
	}
	migratorCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		fmt.Sprintf("linsight config file path, default is %s", defaultServerCfgFile))

	migratorCmd.AddCommand(
		&cobra.Command{
			Use:   "all",
			Short: "starts the server",
			RunE:  runMigration,
		},
	)
	return migratorCmd
}

func runMigration(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		panic(err)
	}
	db, err := dbpkg.NewDB(cfg.Database)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.Error("close db failure when do migration", logger.Error(err))
		}
	}()

	migrator := dbpkg.NewMigrator(db)
	orgUser := dbpkg.NewMigration(&model.OrgUser{})
	orgUser.AddInitRecord(
		&model.OrgUser{OrgID: 1, UserID: 1, Role: accesscontrol.RoleLin},
		&model.OrgUser{OrgID: 1, UserID: 1},
	)
	migrator.AddMigration(orgUser)
	migrator.AddMigration(dbpkg.NewMigration(&model.NotificationSetting{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Notification{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertRule{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Alert{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.UserToken{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Datasource{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Integration{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.IntegrationConnection{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Star{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Preference{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Tag{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.ResourceTag{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Dashboard{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.DashboardProvisioning{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Chart{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Link{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Team{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.TeamMember{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Component{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.OrgComponent{}))
	org := dbpkg.NewMigration(&model.Org{})
	org.AddInitRecord(
		&model.Org{Name: constant.AdminOrgName, UID: uuid.GenerateShortUUID()},
		&model.Org{Name: constant.AdminOrgName},
	)
	migrator.AddMigration(org)

	user := dbpkg.NewMigration(&model.User{})
	salt, _ := util.GetRandomString(10)
	pwd := util.EncodePassword("admin", salt)
	user.AddInitRecord(
		&model.User{Name: "admin", UserName: "admin", Password: pwd, Salt: salt, OrgID: 1, Email: "admin@admin.io"},
		&model.User{UserName: "admin"},
	)
	migrator.AddMigration(user)
	if err = migrator.Run(); err != nil {
		panic(err)
	}
	return nil
}
