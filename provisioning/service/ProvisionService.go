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

package service

import (
	"path/filepath"
	"strings"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/pkg/fileutil"
	"github.com/lindb/linsight/provisioning"
	"github.com/lindb/linsight/provisioning/dashboard"
	depspkg "github.com/lindb/linsight/provisioning/deps"
)

// for testing
var (
	newWatchFn = fileutil.NewWatch
)

// ProvisionType represents provisioning resource type.
type ProvisionType string

const (
	Datasource ProvisionType = "datasources"
	Dashboard  ProvisionType = "dashboards"
)

// ProvisionService represents provisioning service that finds resource based on different provider.
type ProvisionService interface {
	// Run runs resource discovery based on different provider.
	Run() error
	// Shutdown shutdowns all resource discovery.
	Shutdown() error
}

// provisionService implements ProvisionService interface.
type provisionService struct {
	dir   string
	deps  *depspkg.ProvisioningDeps
	watch fileutil.Watch

	watchPaths map[string]provisioning.NewProviderLoader
	logger     logger.Logger
}

// NewProvisionService creates a ProvisionService instance.
func NewProvisionService(deps *depspkg.ProvisioningDeps) ProvisionService {
	return &provisionService{
		dir:  deps.BaseDir,
		deps: deps,
		watchPaths: map[string]provisioning.NewProviderLoader{
			filepath.Join(deps.BaseDir, string(Dashboard)): dashboard.NewDashboardLoader,
		},
		logger: logger.GetLogger("Provisioning", "BackendService"),
	}
}

// Run runs resource discovery based on different provider.
func (p *provisionService) Run() error {
	p.watch = newWatchFn(p.deps.BaseDir, p.handleFileChangeEvent)
	if err := p.watch.Initialize(); err != nil {
		return err
	}
	p.watch.Run()
	return nil
}

// Shutdown shutdowns all resource discovery.
func (p *provisionService) Shutdown() error {
	if p.watch != nil {
		return p.watch.Shutdown()
	}
	// TODO: close provider
	return nil
}

// createProviders creates the prodiver by file config.
func (p *provisionService) createProviders(fileName string, newLoaderFn provisioning.NewProviderLoader) {
	fn, _ := filepath.Abs(fileName)
	loader, err := newLoaderFn(fn, p.deps)
	if err != nil {
		p.logger.Warn("create loader failure", logger.String("file", fn), logger.Error(err))
	} else {
		providers := loader.Load()
		for _, p := range providers {
			p.Run()
		}
	}
}

// handleFileChangeEvent handles file change event.
func (p *provisionService) handleFileChangeEvent(fileName string, op fileutil.Op) {
	if !strings.HasSuffix(fileName, ".yaml") {
		return
	}
	switch op {
	case fileutil.Modify:
		for key, newLoaderFn := range p.watchPaths {
			if strings.HasPrefix(fileName, key) {
				p.createProviders(fileName, newLoaderFn)
			}
		}
	case fileutil.Remove:
		//TODO: remove
	}
}
