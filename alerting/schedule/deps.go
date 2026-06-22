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

package schedule

import (
	"github.com/lindb/linsight/alerting/manager"
	"github.com/lindb/linsight/plugin/datasource"
	"github.com/lindb/linsight/service"
)

// Deps holds all external dependencies required by the alert scheduling subsystem.
type Deps struct {
	Opts            *Options
	DatasourceSrv   service.DatasourceService
	AlertRuleSrv    service.AlertRuleService
	NotificationSrv service.NotificationService
	DatasourceMgr   datasource.Manager

	// AlertMgr receives alert events and handles routing, silencing and notification dispatch.
	AlertMgr manager.AlertManager

	// ManagerDeps exposes the full set of services used by AlertManager, giving DryRunner
	// access to silence / inhibit / route / contact-point data without an active AlertManager.
	ManagerDeps *manager.Deps

	notificationMgr NotificationManager
}
