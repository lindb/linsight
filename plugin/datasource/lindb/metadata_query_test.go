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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_Namespace(t *testing.T) {
	sql, err := buildMetadataQuerySQL(&MetadataQueryRequest{})
	assert.Error(t, err)
	assert.Empty(t, sql)
	sql, err = buildMetadataQuerySQL(&MetadataQueryRequest{Type: Namespace, Where: []Expr{{Key: "namespace", Op: Eq, Value: "ns"}}})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW NAMESPACES WHERE namespace = 'ns'", sql)
}

func TestMetadata_Metric(t *testing.T) {
	sql, err := buildMetadataQuerySQL(&MetadataQueryRequest{Type: Metric, Where: []Expr{{Key: "metric", Op: Eq, Value: "m"}}})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW METRICS WHERE metric = 'm'", sql)

	sql, err = buildMetadataQuerySQL(&MetadataQueryRequest{Type: Metric, Namespace: "ns"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW METRICS ON 'ns'", sql)
}

func TestMetadata_Field(t *testing.T) {
	sql, err := buildMetadataQuerySQL(&MetadataQueryRequest{Type: Field, Metric: "cpu"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW FIELDS FROM 'cpu'", sql)

	sql, err = buildMetadataQuerySQL(&MetadataQueryRequest{Type: Field, Metric: "cpu", Namespace: "ns"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW FIELDS FROM 'cpu' ON 'ns'", sql)
}

func TestMetadata_TagKey(t *testing.T) {
	sql, err := buildMetadataQuerySQL(&MetadataQueryRequest{Type: TagKey, Metric: "cpu"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW TAG KEYS FROM 'cpu'", sql)

	sql, err = buildMetadataQuerySQL(&MetadataQueryRequest{Type: TagKey, Metric: "cpu", Namespace: "ns"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW TAG KEYS FROM 'cpu' ON 'ns'", sql)
}

func TestMetadata_TagValue(t *testing.T) {
	sql, err := buildMetadataQuerySQL(&MetadataQueryRequest{
		Type:   TagValue,
		Metric: "cpu",
		TagKey: "ip",
		Where:  []Expr{{Key: "ip", Op: Like, Value: "1.1.*"}},
	})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW TAG VALUES FROM 'cpu' WITH KEY = 'ip' WHERE ip like '1.1.*'", sql)

	sql, err = buildMetadataQuerySQL(&MetadataQueryRequest{Type: TagValue, Metric: "cpu", TagKey: "ip", Namespace: "ns"})
	assert.NoError(t, err)
	assert.Equal(t, "SHOW TAG VALUES FROM 'cpu' ON 'ns' WITH KEY = 'ip'", sql)
}
