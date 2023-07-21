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
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"github.com/lindb/linsight/model"
	depspkg "github.com/lindb/linsight/provisioning/deps"
	"github.com/lindb/linsight/service"
)

func TestLoader_NewDashboardLoader(t *testing.T) {
	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "read config file failure",
			prepare: func() {
				readFileFn = func(_ string) ([]byte, error) {
					return nil, fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "yaml unmarshal failure",
			prepare: func() {
				readFileFn = func(_ string) ([]byte, error) {
					return []byte("test"), nil
				}
				yamlUnmarshalFn = func(_ []byte, _ interface{}) (err error) {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "create loader successfully",
			prepare: func() {
				readFileFn = func(_ string) ([]byte, error) {
					return []byte("test"), nil
				}
				yamlUnmarshalFn = func(_ []byte, _ interface{}) (err error) {
					return nil
				}
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				readFileFn = os.ReadFile
				yamlUnmarshalFn = yaml.Unmarshal
			}()
			tt.prepare()
			_, err := NewDashboardLoader("test.yaml", nil)
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestLoader_DashboardLoader_Load(t *testing.T) {
	l := &dashboardLoader{
		cfg: &Config{
			Providers: []Provider{{}},
		},
	}
	assert.Len(t, l.Load(), 1)
}

func TestLoader_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgSrv := service.NewMockOrgService(ctrl)
	deps := &depspkg.ProvisioningDeps{
		OrgSrv: orgSrv,
	}
	cases := []struct {
		name     string
		provider *Provider
		prepare  func()
	}{
		{
			name:     "found org failure",
			provider: &Provider{Name: "default", OrgUID: "123"},
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(nil, fmt.Errorf("err"))
			},
		},
		{
			name:     "path is empty",
			provider: &Provider{Name: "default", OrgUID: "123", Type: FileProvider},
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
			},
		},
		{
			name:     "run file load failure",
			provider: &Provider{Name: "default", OrgUID: "123", Type: FileProvider, Options: map[string]any{"path": "../"}},
			prepare: func() {
				orgSrv.EXPECT().GetOrgByUID(gomock.Any(), "123").Return(&model.Org{}, nil)
				fl := NewMockFileLoader(ctrl)
				newFileLoaderFn = func(_ string, _ *model.Org, _ *dashboardProvider) FileLoader {
					return fl
				}
				fl.EXPECT().Run().Return(fmt.Errorf("err"))
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			defer func() {
				newFileLoaderFn = NewFileLoader
			}()
			tt.prepare()
			p := newDashboardProvider("file", tt.provider, deps)
			p.Run()
			assert.Equal(t, "file#default", p.Name())
		})
	}
}
