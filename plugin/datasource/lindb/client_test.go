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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	resty "github.com/go-resty/resty/v2"
	ljson "github.com/lindb/arrow/pkg/json"
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
		jsonUnmarshalFn = func(_ []byte, _ any) error {
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

	restyCli := resty.New()
	restyCli.SetBaseURL("http://127.0.0.1:19999") // non-existent server for unit tests
	cli := &client{
		cfg:      &DatasourceConfig{},
		location: time.Local,
		logger:   logger.GetLogger("Test", "LinDB"),
		cli:      restyCli,
	}

	cases := []struct {
		name    string
		prepare func()
		params  json.RawMessage
		wantErr bool
	}{
		{
			name: "unmarshal query request failure",
			prepare: func() {
				jsonUnmarshalFn = func(_ []byte, _ interface{}) error {
					return fmt.Errorf("err")
				}
			},
			params:  json.RawMessage{},
			wantErr: true,
		},
		{
			// Real unmarshal with Metric set, buildDataQuerySQLFn fails.
			name: "build sql failure",
			prepare: func() {
				buildDataQuerySQLFn = func(_ *DataQueryRequest, _ string, _ string) (string, error) {
					return "", fmt.Errorf("err")
				}
			},
			params:  mustMarshalJSON(DataQueryRequest{Metric: "system.cpu"}),
			wantErr: true,
		},
		{
			// HTTP request fails because there is no server — this is expected in unit tests.
			// The arrow decode path is covered separately in arrow_decoder_test.go.
			name: "http request failure (no server)",
			prepare: func() {
				buildDataQuerySQLFn = func(_ *DataQueryRequest, _ string, _ string) (string, error) {
					return "sql", nil
				}
			},
			params:  mustMarshalJSON(DataQueryRequest{Metric: "system.cpu"}),
			wantErr: true,
		},
	}
	// Test that when LinDB returns a non-200 response, the error message includes the response body.
	t.Run("lindb returns 500 with error body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"internal server error"}`))
		}))
		defer srv.Close()

		mockRestyCli := resty.New()
		mockRestyCli.SetBaseURL(srv.URL)
		c := &client{
			cfg:      &DatasourceConfig{},
			location: time.Local,
			logger:   logger.GetLogger("Test", "LinDB"),
			cli:      mockRestyCli,
		}
		buildDataQuerySQLFn = func(_ *DataQueryRequest, _, _ string) (string, error) {
			return "select * from cpu", nil
		}
		defer func() { buildDataQuerySQLFn = buildDataQuerySQL }()

		_, err := c.DataQuery(context.TODO(), &model.Query{
			Params: mustMarshalJSON(DataQueryRequest{Metric: "system.cpu"}),
		}, model.TimeRange{Start: timeutil.Now()})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
		assert.Contains(t, err.Error(), "internal server error")
	})

	// Test that when LinDB returns a non-200 response with empty body, the error only shows the status code.
	t.Run("lindb returns 500 with empty body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		mockRestyCli := resty.New()
		mockRestyCli.SetBaseURL(srv.URL)
		c := &client{
			cfg:      &DatasourceConfig{},
			location: time.Local,
			logger:   logger.GetLogger("Test", "LinDB"),
			cli:      mockRestyCli,
		}
		buildDataQuerySQLFn = func(_ *DataQueryRequest, _, _ string) (string, error) {
			return "select * from cpu", nil
		}
		defer func() { buildDataQuerySQLFn = buildDataQuerySQL }()

		_, err := c.DataQuery(context.TODO(), &model.Query{
			Params: mustMarshalJSON(DataQueryRequest{Metric: "system.cpu"}),
		}, model.TimeRange{Start: timeutil.Now()})
		assert.Error(t, err)
		assert.EqualError(t, err, "lindb query failed with status 500")
	})

	// Keep the original table-driven sub-tests running.
	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				jsonUnmarshalFn = encoding.JSONUnmarshal
				buildDataQuerySQLFn = buildDataQuerySQL
				buildMetadataQuerySQLFn = buildMetadataQuerySQL
				decodeArrowFn = ljson.DecodeArrowToResultSet
			}()
			tt.prepare()
			_, err := cli.DataQuery(context.TODO(), &model.Query{
				Params: tt.params,
			}, model.TimeRange{Start: timeutil.Now()})
			if tt.wantErr != (err != nil) {
				t.Fatal(tt.name)
			}
		})
	}
}

// mustMarshalJSON serializes v to JSON, panicking on error (test helper only).
func mustMarshalJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return json.RawMessage(data)
}

func TestClient_MetadataQuery(t *testing.T) {
	cli := &client{
		cfg:    &DatasourceConfig{},
		logger: logger.GetLogger("Test", "LinDB"),
	}
	// MetadataQuery is not yet implemented; verify it panics rather than silently failing.
	assert.Panics(t, func() {
		_, _ = cli.MetadataQuery(context.TODO(), &model.Query{
			Params: json.RawMessage{},
		})
	})
}
