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

package lindb

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/timeutil"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin/datasource/lindb/lincli"
)

func TestClient_NewClient(t *testing.T) {
	defer func() {
		jsonUnmarshalFn = encoding.JSONUnmarshal
		loadLocationFn = time.LoadLocation
	}()
	t.Run("create client failure", func(t *testing.T) {
		jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
			return fmt.Errorf("err")
		}
		cli, err := NewClient(&model.Datasource{}, []byte{})
		assert.Error(t, err)
		assert.Nil(t, cli)
	})
	t.Run("load time location failure", func(t *testing.T) {
		jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
			return nil
		}
		loadLocationFn = func(_ string) (*time.Location, error) {
			return nil, fmt.Errorf("err")
		}
		cli, err := NewClient(&model.Datasource{TimeZone: "UTC"}, []byte{})
		assert.Error(t, err)
		assert.Nil(t, cli)
	})
	t.Run("create client successfully", func(t *testing.T) {
		loadLocationFn = time.LoadLocation
		jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
			return nil
		}
		cli, err := NewClient(&model.Datasource{}, []byte{})
		assert.NoError(t, err)
		assert.NotNil(t, cli)
	})
}

func TestClient_DataQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := lincli.NewMockClient(ctrl)
	dq := lincli.NewMockDataQuery(ctrl)
	mockCli.EXPECT().DataQuery().Return(dq).AnyTimes()
	cli := &client{
		cfg:      &DatasourceConfig{},
		location: time.Local,
		logger:   logger.GetLogger("Test", "LinDB"),
		client:   mockCli,
	}

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "unmarshal query request failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "build sql failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildDataQuerySQLFn = func(_ *DataQueryRequest, _ string, _ string) (string, error) {
					return "", fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "data query failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildDataQuerySQLFn = func(_ *DataQueryRequest, _ string, _ string) (string, error) {
					return "sql", nil
				}
				dq.EXPECT().DataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "data query successfully",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildDataQuerySQLFn = func(_ *DataQueryRequest, _ string, _ string) (string, error) {
					return "data_sql", nil
				}
				dq.EXPECT().DataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
				buildDataQuerySQLFn = buildDataQuerySQL
				buildMetadataQuerySQLFn = buildMetadataQuerySQL
			}()
			tt.prepare()
			_, err := cli.DataQuery(context.TODO(), &model.Query{
				Request: json.RawMessage{},
			}, model.TimeRange{From: timeutil.Now()})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestClient_MetadataQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCli := lincli.NewMockClient(ctrl)
	dq := lincli.NewMockDataQuery(ctrl)
	mockCli.EXPECT().DataQuery().Return(dq).AnyTimes()
	cli := &client{
		cfg:    &DatasourceConfig{},
		logger: logger.GetLogger("Test", "LinDB"),
		client: mockCli,
	}

	cases := []struct {
		name    string
		prepare func()
		wantErr bool
	}{
		{
			name: "unmarshal query request failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "build sql failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildMetadataQuerySQLFn = func(_ *MetadataQueryRequest) (string, error) {
					return "", fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "metadata query failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildMetadataQuerySQLFn = func(_ *MetadataQueryRequest) (string, error) {
					return "sql", nil
				}
				dq.EXPECT().MetadataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("err"))
			},
			wantErr: true,
		},
		{
			name: "metadata query successfully",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return nil
				}
				buildMetadataQuerySQLFn = func(_ *MetadataQueryRequest) (string, error) {
					return "metadata_sql", nil
				}
				dq.EXPECT().MetadataQuery(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
				buildDataQuerySQLFn = buildDataQuerySQL
				buildMetadataQuerySQLFn = buildMetadataQuerySQL
			}()
			tt.prepare()
			_, err := cli.MetadataQuery(context.TODO(), &model.Query{
				Request: json.RawMessage{},
			})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}
