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

	"github.com/lindb/common/pkg/logger"
	"github.com/spf13/cobra"

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
	// AlertRuleGroup must be migrated before AlertRule (foreign key group_uid)
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertRuleGroup{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertRule{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Alert{}))
	// new alerting tables (kept separate from legacy tables for smooth migration)
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertEvent{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertRoute{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.ContactPoint{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.ContactPointStatus{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertSilence{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.AlertInhibitRule{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.NotificationTemplate{}))
	// NotificationTemplateExample was removed — examples are now static YAML config (see config/notification_template_examples.yaml).
	// The old table is dropped below after migrator.Run().
	migrator.AddMigration(dbpkg.NewMigration(&model.UserToken{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Datasource{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Integration{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.IntegrationConnection{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Star{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Preference{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Tag{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.ResourceTag{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Dashboard{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.DashboardVersion{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.DashboardProvisioning{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Chart{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Link{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Team{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.TeamMember{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.Menu{}))
	migrator.AddMigration(dbpkg.NewMigration(&model.OrgMenu{}))
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

	// Rename alert_receivers → contact_points if the old table still exists.
	// GORM AutoMigrate creates new tables by struct name; it never renames existing ones.
	rawDB := db.RawDB()
	if rawDB.Migrator().HasTable("alert_receivers") && !rawDB.Migrator().HasTable("contact_points") {
		if err = rawDB.Exec("ALTER TABLE alert_receivers RENAME TO contact_points").Error; err != nil {
			log.Warn("failed to rename alert_receivers to contact_points", logger.Error(err))
		}
	}

	// Drop the legacy 'type' column from notification_templates if it still exists.
	// GORM AutoMigrate never removes columns, so we need to clean it up explicitly.
	if rawDB.Migrator().HasColumn("notification_templates", "type") {
		if err = rawDB.Exec("ALTER TABLE notification_templates DROP COLUMN `type`").Error; err != nil {
			log.Warn("failed to drop legacy type column from notification_templates", logger.Error(err))
		}
	}

	// Copy receiver_uid → contact_point_uid for rows that were created before the rename.
	// AutoMigrate has already added the contact_point_uid column; this back-fills existing data.
	// The UPDATE is idempotent: rows where contact_point_uid is already set are left untouched.
	if rawDB.Migrator().HasColumn("alert_routes", "receiver_uid") {
		if err = rawDB.Exec("UPDATE alert_routes SET contact_point_uid = receiver_uid WHERE (contact_point_uid = '' OR contact_point_uid IS NULL) AND receiver_uid != ''").Error; err != nil {
			log.Warn("failed to copy receiver_uid to contact_point_uid in alert_routes", logger.Error(err))
		}
	}

	// Drop notification_template_examples — examples are now embedded static YAML configuration.
	if rawDB.Migrator().HasTable("notification_template_examples") {
		if err = rawDB.Exec("DROP TABLE notification_template_examples").Error; err != nil {
			log.Warn("failed to drop notification_template_examples table", logger.Error(err))
		}
	}

	// Promote the oldest root-candidate route to is_default=true for each org that has
	// routes but no default root, recovering deployments that contain orphan routes.
	if rawDB.Migrator().HasTable("alert_routes") {
		if err = rawDB.Exec(`
			UPDATE alert_routes SET is_default = 1
			WHERE (parent_uid = '' OR parent_uid IS NULL)
			  AND org_id NOT IN (
			      SELECT DISTINCT org_id FROM alert_routes WHERE is_default = 1
			  )
			  AND id IN (
			      SELECT MIN(id) FROM alert_routes
			      WHERE (parent_uid = '' OR parent_uid IS NULL)
			      GROUP BY org_id
			  )`).Error; err != nil {
			log.Warn("failed to promote default alert route for orgs with no default", logger.Error(err))
		}
	}

	// Re-parent orphan routes (parent_uid="" but is_default=0) to the org's designated root.
	// These arose from the old bug where CreateAlertRoute allowed ParentUID="" without IsDefault.
	// The promotion block above ensures each org has exactly one is_default=1 route.
	if rawDB.Migrator().HasTable("alert_routes") && rawDB.Migrator().HasColumn("alert_routes", "is_default") {
		if err = rawDB.Exec(`
			UPDATE alert_routes SET parent_uid = (
			    SELECT uid FROM (
			        SELECT uid, org_id FROM alert_routes WHERE is_default = 1
			    ) roots WHERE roots.org_id = alert_routes.org_id
			)
			WHERE is_default = 0
			  AND (parent_uid = '' OR parent_uid IS NULL)`).Error; err != nil {
			log.Warn("failed to re-parent orphan alert routes", logger.Error(err))
		}

		// Drop the now-redundant is_default column; ParentUID="" is the sole root marker.
		// SQLite 3.35+ supports DROP COLUMN; older versions will log a warning but continue.
		if err = rawDB.Exec("ALTER TABLE alert_routes DROP COLUMN `is_default`").Error; err != nil {
			log.Warn("failed to drop is_default column from alert_routes", logger.Error(err))
		}
	}

	// Seed root alert route for admin org (org_id=1) on fresh deployments.
	// Existing deployments already have a root (parent_uid='') from the migrations above.
	// Direct DB calls are used instead of AddInitRecord to avoid overwriting an existing root route's UID
	// (which would break child routes that reference it).
	if rawDB.Migrator().HasTable("alert_routes") {
		exists, existErr := db.Exist(&model.AlertRoute{}, "org_id=? and (parent_uid='' or parent_uid is null)", int64(1))
		if existErr != nil {
			log.Warn("failed to check root alert route for admin org", logger.Error(existErr))
		} else if !exists {
			rootRoute := model.NewDefaultPolicyRoute(1, uuid.GenerateShortUUID(), 1)
			if err = db.Create(rootRoute); err != nil {
				log.Warn("failed to create root alert route for admin org", logger.Error(err))
			}
		}
	}
	return nil
}
