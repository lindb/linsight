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
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/provisioning"
	depspkg "github.com/lindb/linsight/provisioning/deps"
)

// for testing
var (
	readFileFn      = os.ReadFile
	yamlUnmarshalFn = yaml.Unmarshal
	newFileLoaderFn = NewFileLoader
)

// dashboardLoader implements ProviderLoader interface.
type dashboardLoader struct {
	fileName string
	cfg      *Config
	deps     *depspkg.ProvisioningDeps
}

// NewDashboardLoader creates dashboard provider loader.
func NewDashboardLoader(fileName string, deps *depspkg.ProvisioningDeps) (provisioning.ProviderLoader, error) {
	content, err := readFileFn(fileName)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yamlUnmarshalFn(content, &config)
	if err != nil {
		return nil, err
	}
	return &dashboardLoader{
		fileName: fileName,
		cfg:      &config,
		deps:     deps,
	}, err
}

// Load returns dashboard provider from config file.
func (l *dashboardLoader) Load() (rs []provisioning.Provider) {
	for _, cfg := range l.cfg.Providers {
		rs = append(rs, newDashboardProvider(l.fileName, &cfg, l.deps))
	}
	return
}

// dashboardProvider implements Provider interface.
type dashboardProvider struct {
	id   string
	cfg  *Provider
	deps *depspkg.ProvisioningDeps

	logger logger.Logger
}

// newDashboardProvider creates dashboard provider instance.
func newDashboardProvider(cfgFile string, cfg *Provider, deps *depspkg.ProvisioningDeps) provisioning.Provider {
	return &dashboardProvider{
		id:     cfgFile + "#" + cfg.Name,
		cfg:    cfg,
		deps:   deps,
		logger: logger.GetLogger("Provisioning", "DashboardProvider"),
	}
}

// Name returns dashboard provider's identification.
func (p *dashboardProvider) Name() string {
	return p.id
}

// Run runs dashboard discovery.
func (p *dashboardProvider) Run() {
	// get org. info
	org, err := p.deps.OrgSrv.GetOrgByUID(context.TODO(), p.cfg.OrgUID)
	if err != nil {
		p.logger.Warn("get org failure", logger.String("provider", p.id), logger.Error(err))
		return
	}
	if p.cfg.Type == FileProvider {
		dirName, ok := p.cfg.Options["path"]
		if !ok {
			p.logger.Warn("load path is empty, ignore", logger.String("provider", p.id))
			return
		}
		dir, _ := filepath.Abs(fmt.Sprintf("%v", dirName))
		fl := newFileLoaderFn(dir, org, p)
		if err := fl.Run(); err != nil {
			p.logger.Warn("run file loader failure", logger.String("provider", p.id), logger.Error(err))
			return
		}
		// TODO: add shuwdown
	}
}
