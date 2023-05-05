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

package datasource

import (
	"encoding/json"

	"github.com/lindb/common/pkg/encoding"

	"github.com/lindb/linsight/model"

	"github.com/go-resty/resty/v2"
)

type LinDBConfig struct {
	Database string `json:"database"`
}

type LinDBRequest struct {
	SQL string `json:"sql"`
}

type LinDBHandle struct {
	url string
	cfg json.RawMessage
}

func NewLinDBHandler(url string, cfg json.RawMessage) Handler {
	return &LinDBHandle{
		url: url,
		cfg: cfg,
	}
}

func (handle *LinDBHandle) DataQuery(query *model.Query, timeRange model.Range) (any, error) {
	cli := resty.New()
	r := cli.R()
	r.Header.Set("Content-Type", "application/json")
	config := LinDBConfig{}
	cfgData, _ := handle.cfg.MarshalJSON()
	if err := encoding.JSONUnmarshal(cfgData, &config); err != nil {
		return nil, err
	}
	data, _ := query.Request.MarshalJSON()
	req := LinDBRequest{}
	if err := encoding.JSONUnmarshal(data, &req); err != nil {
		return nil, err
	}
	resp, err := r.SetBody(map[string]any{
		"db":  config.Database,
		"sql": req.SQL,
	}).Put(handle.url + "/api/v1/exec")
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resp.Body()), nil
}
