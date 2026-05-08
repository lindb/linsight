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
	"errors"
	"fmt"
	"net/http"
	"time"

	resty "github.com/go-resty/resty/v2"
	ljson "github.com/lindb/arrow/pkg/json"
	lincli "github.com/lindb/client_go"
	"github.com/lindb/common/pkg/encoding"
	"github.com/lindb/common/pkg/logger"
	"github.com/lindb/common/pkg/timeutil"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
)

// decodeArrowFn is the Arrow IPC → ResultSet decoder, exposed as a variable
// so tests can substitute a mock without changing call sites.
var decodeArrowFn = ljson.DecodeArrowToResultSet

// contentTypeArrow is the MIME type for Apache Arrow IPC stream format.
const contentTypeArrow = "application/vnd.apache.arrow.stream"

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
	cli       *resty.Client
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
	cli := resty.New()
	cli.SetBaseURL(datasource.URL)

	return &client{
		cfg:       config,
		datasouce: datasource,
		location:  location,
		cli:       cli,
		client:    lincli.NewClient(datasource.URL),
		logger:    logger.GetLogger("DatasourcePlugin", "LinDB"),
	}, nil
}

// DataQuery queries metric data.
func (cli *client) DataQuery(ctx context.Context, req *model.Query, timeRange model.TimeRange) (any, error) {
	data, err := req.Params.MarshalJSON()
	if err != nil {
		return nil, errors.New("data query: marshal params: " + err.Error())
	}
	dataQueryReq := DataQueryRequest{}
	var sql string
	database := cli.cfg.Database
	if err0 := jsonUnmarshalFn(data, &dataQueryReq); err0 != nil {
		return nil, err0
	}
	if dataQueryReq.SQL != "" {
		sql = dataQueryReq.SQL
	}
	if dataQueryReq.Database != "" {
		database = dataQueryReq.Database
	}

	if dataQueryReq.SQL == "" && dataQueryReq.Metric != "" {
		sql, err = buildDataQuerySQLFn(&dataQueryReq, cli.formatTime(timeRange.Start), cli.formatTime(timeRange.End))
		if err != nil {
			return nil, err
		}
	}
	if sql == "" {
		cli.logger.Warn("sql is empty", logger.String("params", string(data)))
		return nil, nil
	}
	// send request
	resp, err := cli.cli.R().
		SetBody(map[string]any{"sql": sql, "db": database, "timeRange": timeRange}).
		SetHeader("Accept", contentTypeArrow).
		Put("/api/v1/exec")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		body := resp.Body()
		if len(body) == 0 {
			return nil, errors.New("no data found")
		}
		// Decode Arrow IPC stream and convert to the standard ResultSet JSON format.
		rs, decodeErr := decodeArrowFn(body)
		if decodeErr != nil {
			return nil, decodeErr
		}
		// rs is nil only when body is empty, which is already guarded above.
		// Retained as a safety net against future changes to decodeArrowFn.
		if rs == nil {
			return nil, errors.New("no data found")
		}
		rsJSON, marshalErr := json.Marshal(rs)
		if marshalErr != nil {
			return nil, marshalErr
		}
		return json.RawMessage(rsJSON), nil
	}
	// Include the response body in the error so callers can see the actual LinDB error message.
	if body := resp.Body(); len(body) > 0 {
		return nil, fmt.Errorf("lindb query failed with status %d: %s", resp.StatusCode(), body)
	}
	return nil, fmt.Errorf("lindb query failed with status %d", resp.StatusCode())
}

// MetadataQuery queries metric metadata.
func (cli *client) MetadataQuery(ctx context.Context, req *model.Query) (any, error) {
	data, err := req.Params.MarshalJSON()
	if err != nil {
		return nil, errors.New("metadata query: marshal params: " + err.Error())
	}
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
