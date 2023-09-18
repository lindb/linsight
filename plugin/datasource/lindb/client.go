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
	"time"

	lincli "github.com/lindb/client_go"
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/timeutil"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

//go:generate mockgen -destination=./lincli/client_mock.go -package=lincli github.com/lindb/client_go Client
//go:generate mockgen -destination=./lincli/dataquery_mock.go -package=lincli github.com/lindb/client_go/api DataQuery

// for testing
var (
	jsonUnmarshalFn         = encoding.JSONUnmarshal
	buildDataQuerySQLFn     = buildDataQuerySQL
	buildMetadataQuerySQLFn = buildMetadataQuerySQL
	loadLocationFn          = time.LoadLocation
)

// client implements plugin.DatasourcePlugin for LinDB.
type client struct {
	datasouce *model.Datasource
	cfg       *DatasourceConfig
	client    lincli.Client
	location  *time.Location

	logger logger.Logger
}

// NewClient creates a LinDB client.
func NewClient(datasource *model.Datasource, cfg json.RawMessage) (p plugin.DatasourcePlugin, err error) {
	config := &DatasourceConfig{}
	cfgData, _ := cfg.MarshalJSON()
	if err0 := jsonUnmarshalFn(cfgData, config); err0 != nil {
		return nil, err0
	}
	var location *time.Location
	if datasource.TimeZone == "" {
		location = time.Local
	} else {
		location, err = loadLocationFn(datasource.TimeZone)
		if err != nil {
			return nil, err
		}
	}

	return &client{
		cfg:       config,
		datasouce: datasource,
		location:  location,
		client:    lincli.NewClient(datasource.URL),
		logger:    logger.GetLogger("DatasourcePlugin", "LinDB"),
	}, nil
}

// DataQuery queries metric data.
func (cli *client) DataQuery(ctx context.Context, req *model.Query, timeRange model.TimeRange) (any, error) {
	data, _ := req.Request.MarshalJSON()
	dataQueryReq := &DataQueryRequest{}
	if err := jsonUnmarshalFn(data, &dataQueryReq); err != nil {
		return nil, err
	}
	sql, err := buildDataQuerySQLFn(dataQueryReq, cli.formatTime(timeRange.From), cli.formatTime(timeRange.To))
	if err != nil {
		return nil, err
	}
	query := cli.client.DataQuery()
	rs, err := query.DataQuery(ctx, cli.cfg.Database, sql)
	if err != nil {
		return nil, err
	}
	cli.logger.Info("data query", logger.String("database", cli.cfg.Database), logger.String("sql", sql))
	return rs, nil
}

// MetadataQuery queries metric metadata.
func (cli *client) MetadataQuery(ctx context.Context, req *model.Query) (any, error) {
	data, _ := req.Request.MarshalJSON()
	dataQueryReq := &MetadataQueryRequest{}
	if err := jsonUnmarshalFn(data, &dataQueryReq); err != nil {
		return nil, err
	}

	sql, err := buildMetadataQuerySQLFn(dataQueryReq)
	if err != nil {
		return nil, err
	}
	query := cli.client.DataQuery()
	rs, err := query.MetadataQuery(ctx, cli.cfg.Database, sql)
	if err != nil {
		return nil, err
	}
	cli.logger.Info("metadata query", logger.String("database", cli.cfg.Database), logger.String("sql", sql))
	return rs, nil
}

func (cli *client) formatTime(timestamp int64) string {
	if timestamp <= 0 {
		return ""
	}
	ts := time.UnixMilli(timestamp)
	return ts.In(cli.location).Format(timeutil.DataTimeFormat2)
}
