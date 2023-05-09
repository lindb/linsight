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

import "fmt"

func buildMetadataQuerySQL(req *MetadataQueryRequest) string {
	switch req.Type {
	case Namespace:
		return "show namespaces"
	case Metric:
		return "show metrics" + buildMetricNamePrefix(req.Prefix)
	case Field:
		return fmt.Sprintf("show fields from '%s'", req.Metric)
	case TagKey:
		return fmt.Sprintf("show tag keys from '%s'", req.Metric)
	case TagValue:
	}
	return ""
}

func buildMetricNamePrefix(prefix string) string {
	if prefix != "" {
		return fmt.Sprintf(" where metric='%s'", prefix)
	}
	return ""
}
