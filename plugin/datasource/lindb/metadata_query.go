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
	"bytes"
	"fmt"
)

// buildMetadataQuerySQL returns metadata LinDB query language.
func buildMetadataQuerySQL(req *MetadataQueryRequest) (string, error) {
	return NewBuilder(req.Type).Namespace(req.Namespace).Metric(req.Metric).TagKey(req.TagKey).Prefix(req.Prefix).ToSQL()
}

// MetadataQueryBuilder represents LinDB query language builder.
type MetadataQueryBuilder struct {
	queryType MetadataType
	namespace string
	metric    string
	tagKey    string
	prefix    string
}

func NewBuilder(queryType MetadataType) *MetadataQueryBuilder {
	return &MetadataQueryBuilder{
		queryType: queryType,
	}
}

// Metric sets metric name.
func (b *MetadataQueryBuilder) Metric(metric string) *MetadataQueryBuilder {
	b.metric = metric
	return b
}

// Namespace sets namespace.
func (b *MetadataQueryBuilder) Namespace(namespace string) *MetadataQueryBuilder {
	b.namespace = namespace
	return b
}

// Prefix sets search prefix.
func (b *MetadataQueryBuilder) Prefix(prefix string) *MetadataQueryBuilder {
	b.prefix = prefix
	return b
}

// TagKey sets tag key.
func (b *MetadataQueryBuilder) TagKey(tagKey string) *MetadataQueryBuilder {
	b.tagKey = tagKey
	return b
}

// ToSQL returns the LinDB query language based on params.
func (b *MetadataQueryBuilder) ToSQL() (sql string, err error) {
	if b.queryType == "" {
		err = fmt.Errorf("query type is required")
		return
	}

	sqlBuf := &bytes.Buffer{}
	switch b.queryType {
	case Namespace:
		sqlBuf.WriteString("SHOW NAMESPACES")
	case Metric:
		sqlBuf.WriteString("SHOW METRICS")
		b.buildNamespace(sqlBuf)
	case Field:
		sqlBuf.WriteString("SHOW FIELDS")
		sqlBuf.WriteString(" FROM ")
		fmt.Fprintf(sqlBuf, "'%s'", b.metric)
		b.buildNamespace(sqlBuf)
	case TagKey:
		sqlBuf.WriteString("SHOW TAG KEYS")
		sqlBuf.WriteString(" FROM ")
		fmt.Fprintf(sqlBuf, "'%s'", b.metric)
		b.buildNamespace(sqlBuf)
	case TagValue:
		sqlBuf.WriteString("SHOW TAG VALUES")
		sqlBuf.WriteString(" FROM ")
		fmt.Fprintf(sqlBuf, "'%s'", b.metric)
		b.buildNamespace(sqlBuf)
		sqlBuf.WriteString(" WITH KEY =")
		fmt.Fprintf(sqlBuf, " '%s'", b.tagKey)
	}
	return sqlBuf.String(), nil
}

func (b *MetadataQueryBuilder) buildNamespace(sqlBuf *bytes.Buffer) {
	if len(b.namespace) > 0 {
		sqlBuf.WriteString(" ON ")
		fmt.Fprintf(sqlBuf, "'%s'", b.namespace)
	}
}
