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

package api

import (
	"github.com/gin-gonic/gin"

	httppkg "github.com/lindb/common/pkg/http"

	depspkg "github.com/lindb/linsight/http/deps"
)

// TagAPI represents tag related api handlers.
type TagAPI struct {
	deps *depspkg.API
}

// NewTagAPI creates a TagAPI instance.
func NewTagAPI(deps *depspkg.API) *TagAPI {
	return &TagAPI{
		deps: deps,
	}
}

// FindTags returns tag list by given term prefix.
func (api *TagAPI) FindTags(c *gin.Context) {
	term := c.Query("term")
	tags, err := api.deps.TagSrv.FindTags(c.Request.Context(), term)
	if err != nil {
		httppkg.Error(c, err)
		return
	}
	httppkg.OK(c, tags)
}
