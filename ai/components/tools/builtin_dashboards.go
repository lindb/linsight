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

package tools

import (
	"context"
	"encoding/json"

	"github.com/lindb/linsight/model"
)

// maxDashboardResults caps the number of dashboards returned to the LLM.
const maxDashboardResults = 20

// searchDashboardsInput is the input struct for linsight_search_dashboards.
type searchDashboardsInput struct {
	Title string   `json:"title"  jsonschema:"description=Filter dashboards by title (partial match)."`
	Tags  []string `json:"tags"   jsonschema:"description=Filter dashboards that have all of the given tags."`
	Limit int      `json:"limit"  jsonschema:"description=Maximum number of results to return (1-50). Default: 20."`
}

// dashboardInfo is the per-dashboard summary returned by linsight_search_dashboards.
type dashboardInfo struct {
	UID   string   `json:"uid"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

// searchDashboards calls the service and maps the results to the LLM-friendly shape.
func searchDashboards(ctx context.Context, srv dashboardSearcher, input searchDashboardsInput) ([]dashboardInfo, error) {
	limit := input.Limit
	if limit <= 0 || limit > 50 {
		limit = maxDashboardResults
	}

	req := &model.SearchDashboardRequest{
		PagingParam: model.PagingParam{Limit: limit},
		Title:       input.Title,
		Tags:        input.Tags,
	}

	dashboards, _, err := srv.SearchDashboards(ctx, req)
	if err != nil {
		return nil, err
	}

	result := make([]dashboardInfo, 0, len(dashboards))
	for _, d := range dashboards {
		var tags []string
		if len(d.Tags) > 0 {
			_ = json.Unmarshal(d.Tags, &tags)
		}
		result = append(result, dashboardInfo{
			UID:   d.UID,
			Title: d.Title,
			Tags:  tags,
		})
	}
	return result, nil
}
