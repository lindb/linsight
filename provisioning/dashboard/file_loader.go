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

package dashboard

import (
	"context"
	"strings"

	"github.com/lindb/common/pkg/logger"
	"gorm.io/datatypes"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/fileutil"
)

//go:generate mockgen -source=./file_loader.go -destination=./file_loader_mock.go -package=dashboard

var (
	newWatchFn = fileutil.NewWatch
)

// FileLoader represents dashboard file loader.
type FileLoader interface {
	// Run runs file loader, discovery dashboard from local file system.
	Run() error
	// Shutdown shutdowns file loader
	Shutdown() error
}

type fileLoader struct {
	dir      string
	watch    fileutil.Watch
	org      *model.Org
	provider *dashboardProvider

	logger logger.Logger
}

// NewFileLoader creates a FileLoader.
func NewFileLoader(dir string, org *model.Org, provider *dashboardProvider) FileLoader {
	return &fileLoader{
		dir:      dir,
		org:      org,
		provider: provider,
		logger:   logger.GetLogger("Provisioning", "DashboardFileLoader"),
	}
}

// Run runs file loader, discovery dashboard from local file system.
func (fl *fileLoader) Run() error {
	fl.watch = newWatchFn(fl.dir, fl.handleFileChangeEvent)
	if err := fl.watch.Initialize(); err != nil {
		return err
	}
	fl.watch.Run()
	return nil
}

// Shutdown shutdowns file loader
func (fl *fileLoader) Shutdown() error {
	if fl.watch != nil {
		return fl.watch.Shutdown()
	}
	return nil
}

// saveDashboard saves dashboard which found from local file system.
func (fl *fileLoader) saveDashboard(fileName string) {
	if !strings.HasSuffix(fileName, ".json") {
		return
	}
	p := fl.provider
	content, err := readFileFn(fileName)
	if err != nil {
		fl.logger.Warn("read dashboard data from file failure",
			logger.String("provider", p.id), logger.String("file", fileName),
			logger.Error(err))
		return
	}
	if len(content) == 0 {
		// ignore if file content is empty
		return
	}
	if err := fl.provider.deps.DashboardSrv.SaveProvisioningDashboard(context.TODO(), &model.SaveProvisioningDashboardRequest{
		Org: fl.org,
		Dashboard: &model.Dashboard{
			OrgID:  fl.org.ID,
			Config: datatypes.JSON(content),
		},
		Provisioning: &model.DashboardProvisioning{
			OrgID:          fl.org.ID,
			Name:           fileName,
			AllowUIUpdates: p.cfg.AllowUIUpdates,
			External:       p.id,
			Checksum:       "",
		},
	}); err != nil {
		fl.logger.Warn("save dashboard failure", logger.String("provider", p.id), logger.Error(err))
	}
}

// removeDashboard removes dashboard which not exist in local file system.
func (fl *fileLoader) removeDashboard(fileName string) {
	if !strings.HasSuffix(fileName, ".json") {
		return
	}
	if err := fl.provider.deps.DashboardSrv.RemoveProvisioningDashboard(context.TODO(), &model.RemoveProvisioningDashboardRequest{
		Org:      fl.org,
		Name:     fileName,
		External: fl.provider.id,
	}); err != nil {
		fl.logger.Warn("remove dashboard failure", logger.String("provider", fl.provider.id), logger.Error(err))
	}
}

// handleFileChangeEvent handles file change event.
func (fl *fileLoader) handleFileChangeEvent(fileName string, op fileutil.Op) {
	switch op {
	case fileutil.Modify:
		fl.saveDashboard(fileName)
	case fileutil.Remove:
		fl.removeDashboard(fileName)
	}
}
