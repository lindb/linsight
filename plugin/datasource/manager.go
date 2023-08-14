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
	"fmt"

	"github.com/lindb/linsight/model"
	"github.com/lindb/linsight/plugin"
	"github.com/lindb/linsight/plugin/datasource/lindb"
	"github.com/lindb/linsight/plugin/datasource/lingo"
)

//go:generate mockgen -source=./manager.go -destination=./manager_mock.go -package=datasource

var datasourceClients = make(map[string]plugin.NewDatasourcePlugin)

func init() {
	datasourceClients[model.LinDBDatasource] = lindb.NewClient
	datasourceClients[model.LinGoDatasource] = lingo.NewClient
}

// Manager represents datasouce plugin manager.
type Manager interface {
	// GetPlugin returns datasource plugin by type.
	GetPlugin(datasouce *model.Datasource) (plugin.DatasourcePlugin, error)
}

// manager implements Manager interface.
type manager struct {
}

// NewDatasourceManager creates datasource Manager instance.
func NewDatasourceManager() Manager {
	return &manager{}
}

// GetPlugin returns datasource plugin by type.
func (mgr *manager) GetPlugin(datasource *model.Datasource) (plugin.DatasourcePlugin, error) {
	newCliFn, ok := datasourceClients[datasource.Type]
	if !ok {
		return nil, fmt.Errorf("datasouce not support, type: %s", datasource.Type)
	}
	// FIXME: add cache
	return newCliFn(datasource, json.RawMessage(datasource.Config))
}
