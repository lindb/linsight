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

	lincli "github.com/lindb/client_go"
	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

type client struct {
	cfg    *DatasourceConfig
	client lincli.Client
}

func NewClient(url string, cfg json.RawMessage) (plugin.DatasourcePlugin, error) {
	config := &DatasourceConfig{}
	cfgData, _ := cfg.MarshalJSON()
	if err := encoding.JSONUnmarshal(cfgData, config); err != nil {
		return nil, err
	}
	return &client{
		cfg:    config,
		client: lincli.NewClient(url),
	}, nil
}

func (cli *client) DataQuery(ctx context.Context, req *model.Query, timeRange model.TimeRange) (any, error) {
	data, _ := req.Request.MarshalJSON()
	dataQueryReq := &DataQueryRequest{}
	if err := encoding.JSONUnmarshal(data, &dataQueryReq); err != nil {
		return nil, err
	}

	sql, err := buildDataQuerySQL(dataQueryReq, timeRange)
	if err != nil {
		return nil, err
	}
	query := cli.client.DataQuery()
	rs, err := query.DataQuery(ctx, cli.cfg.Database, sql)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func (cli *client) MetadataQuery(ctx context.Context, req *model.Query) (any, error) {
	data, _ := req.Request.MarshalJSON()
	fmt.Println(string(data))
	dataQueryReq := &MetadataQueryRequest{}
	if err := encoding.JSONUnmarshal(data, &dataQueryReq); err != nil {
		return nil, err
	}

	fmt.Println(dataQueryReq)
	sql, err := buildMetadataQuerySQL(dataQueryReq)
	if err != nil {
		return nil, err
	}
	fmt.Println(sql)
	query := cli.client.DataQuery()
	rs, err := query.MetadataQuery(ctx, cli.cfg.Database, sql)
	if err != nil {
		return nil, err
	}
	return rs, nil
}
