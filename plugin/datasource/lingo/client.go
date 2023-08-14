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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

//go:generate mockgen -source=./client.go -destination=./client_mock.go -package=lingo

// HTTPClient represents http client interface.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// for testing
var (
	jsonUnmarshalFn = encoding.JSONUnmarshal
	newRequestFn    = http.NewRequestWithContext
	readAllFn       = io.ReadAll
)

// client implements plugin.DatasourcePlugin for LinGo.
type client struct {
	datasouce *model.Datasource
	cfg       *DatasourceConfig
	httpCli   HTTPClient

	logger logger.Logger
}

// NewClient creates a LinGo client.
func NewClient(datasource *model.Datasource, cfg json.RawMessage) (p plugin.DatasourcePlugin, err error) {
	config := &DatasourceConfig{}
	cfgData, _ := cfg.MarshalJSON()
	if err0 := jsonUnmarshalFn(cfgData, config); err0 != nil {
		return nil, err0
	}

	return &client{
		cfg:       config,
		datasouce: datasource,
		httpCli: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10, // TODO: need add config
				IdleConnTimeout:     30 * time.Second,
				MaxIdleConnsPerHost: 5,
			},
		},
		logger: logger.GetLogger("DatasourcePlugin", "LinDB"),
	}, nil
}

// DataQuery queries trace data from LinGo.
func (cli *client) DataQuery(ctx context.Context, req *model.Query, timeRange model.TimeRange) (any, error) {
	data, _ := req.Request.MarshalJSON()
	traceQueryReq := &GetTraceRequest{}
	if err := jsonUnmarshalFn(data, &traceQueryReq); err != nil {
		return nil, err
	}
	httpReq, err := newRequestFn(ctx, "GET",
		fmt.Sprintf("%s?pipeline=%s&traceId=%s", cli.datasouce.URL, cli.cfg.Pipeline, traceQueryReq.TraceID),
		nil)
	if err != nil {
		return nil, err
	}
	resp, err := cli.httpCli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	d, err := readAllFn(resp.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(d), nil
}

func (cli *client) MetadataQuery(ctx context.Context, req *model.Query) (any, error) {
	panic("not support")
}
