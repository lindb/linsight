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
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/pkg/fileutil"
	"github.com/lindb/linsight/provisioning"
	"github.com/lindb/linsight/provisioning/deps"
)

func TestProvisionService_Run(t *testing.T) {
	ps := NewProvisionService(&deps.ProvisioningDeps{})
	assert.NoError(t, ps.Shutdown())
	ctrl := gomock.NewController(t)
	defer func() {
		newWatchFn = fileutil.NewWatch
		ctrl.Finish()
	}()
	watch := fileutil.NewMockWatch(ctrl)
	newWatchFn = func(_ string, _ func(fileName string, op fileutil.Op)) fileutil.Watch {
		return watch
	}
	watch.EXPECT().Initialize().Return(fmt.Errorf("err"))
	assert.Error(t, ps.Run())

	watch.EXPECT().Initialize().Return(nil)
	watch.EXPECT().Run()
	assert.NoError(t, ps.Run())
	watch.EXPECT().Shutdown().Return(nil)
	assert.NoError(t, ps.Shutdown())
}

func TestProvisionService_handleFileChangeEvent(t *testing.T) {
	f, _ := os.Create("dashboards/a.yaml")
	defer func() {
		_ = f.Close()
		os.RemoveAll("dashboards")
	}()
	ps := NewProvisionService(&deps.ProvisioningDeps{
		BaseDir: "./",
	})
	ps1 := ps.(*provisionService)
	cases := []struct {
		name     string
		fileName string
		op       fileutil.Op
		prepare  func()
	}{
		{
			name:     "remove file",
			fileName: "a.yaml",
			op:       fileutil.Remove,
		},
		{
			name:     "not yaml file",
			fileName: "a.json",
			op:       fileutil.Remove,
		},
		{
			name:     "not macth resource type",
			fileName: "dd/a.yaml",
			op:       fileutil.Modify,
		},
		{
			name:     "not macth resource type",
			fileName: "dashboards/a.yaml",
			op:       fileutil.Modify,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}
			ps1.handleFileChangeEvent(tt.fileName, tt.op)
		})
	}
}

func TestProvisionService_createProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	ps := NewProvisionService(&deps.ProvisioningDeps{})
	ps1 := ps.(*provisionService)
	loader := provisioning.NewMockProviderLoader(ctrl)

	provider := provisioning.NewMockProvider(ctrl)
	loader.EXPECT().Load().Return([]provisioning.Provider{provider})
	provider.EXPECT().Run()

	ps1.createProviders("a.yaml", func(fileName string, deps *deps.ProvisioningDeps) (provisioning.ProviderLoader, error) {
		return loader, nil
	})
}
