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

package lingo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/timeutil"
	"github.com/stretchr/testify/assert"

	"github.com/lindb/linsight/model"
)

func TestClient_NewClient(t *testing.T) {
	defer func() {
		jsonUnmarshalFn = encoding.JSONUnmarshal
	}()
	t.Run("create client failure", func(t *testing.T) {
		jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
			return fmt.Errorf("err")
		}
		cli, err := NewClient(&model.Datasource{}, []byte{})
		assert.Error(t, err)
		assert.Nil(t, cli)
	})
	t.Run("create client successfully", func(t *testing.T) {
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

	httpCli := NewMockHTTPClient(ctrl)
	cli := &client{
		cfg:       &DatasourceConfig{},
		datasouce: &model.Datasource{},
		httpCli:   httpCli,
		logger:    logger.GetLogger("Test", "LinGo"),
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
			name: "new http request failure",
			prepare: func() {
				newRequestFn = func(_ context.Context, _, _ string, _ io.Reader) (*http.Request, error) {
					return nil, fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "do http request failure",
			prepare: func() {
				httpCli.EXPECT().Do(gomock.Any()).DoAndReturn(func(_ *http.Request) (*http.Response, error) {
					return nil, fmt.Errorf("err")
				})
			},
			wantErr: true,
		},
		{
			name: "read resp body failure",
			prepare: func() {
				httpCli.EXPECT().Do(gomock.Any()).DoAndReturn(func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				})
				readAllFn = func(_ io.Reader) ([]byte, error) {
					return nil, fmt.Errorf("err")
				}
			},
			wantErr: true,
		},
		{
			name: "get trace data successfully",
			prepare: func() {
				httpCli.EXPECT().Do(gomock.Any()).DoAndReturn(func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				})
				readAllFn = func(_ io.Reader) ([]byte, error) {
					return nil, nil
				}
			},
		},
	}
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
				newRequestFn = http.NewRequestWithContext
				readAllFn = io.ReadAll
			}()
			tt.prepare()
			_, err := cli.DataQuery(context.TODO(), &model.Query{
				Request: json.RawMessage([]byte(`{}`)),
			}, model.TimeRange{From: timeutil.Now()})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

func TestClient_MetadataQuery(t *testing.T) {
	assert.Panics(t, func() {
		cli, err := NewClient(&model.Datasource{}, []byte(`{}`))
		assert.NoError(t, err)
		_, _ = cli.MetadataQuery(context.TODO(), nil)
		assert.True(t, false)
	})
}
