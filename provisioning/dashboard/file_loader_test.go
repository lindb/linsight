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

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/pkg/fileutil"
	depspkg "github.com/lindb/linsight/provisioning/deps"
	"github.com/lindb/linsight/service"
)

func TestFileLoader_Run(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer func() {
		newWatchFn = fileutil.NewWatch
		ctrl.Finish()
	}()

	w := fileutil.NewMockWatch(ctrl)
	newWatchFn = func(_ string, _ func(fileName string, op fileutil.Op)) fileutil.Watch {
		return w
	}

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "init watch failure",
			prepare: func() {
				w.EXPECT().Initialize().Return(fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "run successfully",
			prepare: func() {
				w.EXPECT().Initialize().Return(nil)
				w.EXPECT().Run()
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			fl := NewFileLoader("test.yaml", nil, nil)
			tt.prepare()
			err := fl.Run()
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestFileLoader_Shutdonw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	w := fileutil.NewMockWatch(ctrl)
	fl := &fileLoader{
		watch: w,
	}

	w.EXPECT().Shutdown().Return(nil)
	assert.Nil(t, fl.Shutdown())
	fl.watch = nil
	assert.Nil(t, fl.Shutdown())
}

func TestFileLoader_handeFileChangeEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dashboardSrv := service.NewMockDashboardService(ctrl)
	deps := &depspkg.ProvisioningDeps{
		DashboardSrv: dashboardSrv,
	}
	provider := &dashboardProvider{
		deps: deps,
		cfg:  &Provider{},
	}
	cases := []struct {
		name     string
		fileName string
		op       fileutil.Op
		prepare  func()
	}{
		{
			name:     "not json file",
			op:       fileutil.Modify,
			fileName: "a.yaml",
		},
		{
			name:     "read json file failure",
			op:       fileutil.Modify,
			fileName: "d.json",
			prepare: func() {
				readFileFn = func(name string) ([]byte, error) {
					return nil, fmt.Errorf("err")
				}
			},
		},
		{
			name:     "empty json file",
			op:       fileutil.Modify,
			fileName: "d.json",
			prepare: func() {
				readFileFn = func(name string) ([]byte, error) {
					return nil, nil
				}
			},
		},
		{
			name:     "save dashboard failure",
			op:       fileutil.Modify,
			fileName: "d.json",
			prepare: func() {
				readFileFn = func(name string) ([]byte, error) {
					return []byte(`{}`), nil
				}
				dashboardSrv.EXPECT().SaveProvisioningDashboard(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
		},
		{
			name:     "save dashboard successfully",
			op:       fileutil.Modify,
			fileName: "d.json",
			prepare: func() {
				readFileFn = func(name string) ([]byte, error) {
					return []byte(`{}`), nil
				}
				dashboardSrv.EXPECT().SaveProvisioningDashboard(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name:     "not json file",
			op:       fileutil.Remove,
			fileName: "a.yaml",
		},
		{
			name:     "remove dashboard failure",
			op:       fileutil.Remove,
			fileName: "d.json",
			prepare: func() {
				dashboardSrv.EXPECT().RemoveProvisioningDashboard(gomock.Any(), gomock.Any()).Return(fmt.Errorf("err"))
			},
		},
		{
			name:     "remove dashboard successfully",
			op:       fileutil.Remove,
			fileName: "d.json",
			prepare: func() {
				dashboardSrv.EXPECT().RemoveProvisioningDashboard(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(_ *testing.T) {
			defer func() {
				readFileFn = os.ReadFile
			}()
			fl := NewFileLoader("test.yaml",
				&model.Org{BaseModel: model.BaseModel{ID: 123}}, provider)
			fl1 := fl.(*fileLoader)
			if tt.prepare != nil {
				tt.prepare()
			}
			fl1.handleFileChangeEvent(tt.fileName, tt.op)
		})
	}
}
