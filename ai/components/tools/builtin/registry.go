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

package builtin

import "github.com/cloudwego/eino/components/tool"

// BuiltinDeps carries all service dependencies that builtin tool constructors may need.
// Each constructor only reads the fields it requires; unused fields are ignored.
type BuiltinDeps struct {
	DatasourceSrv datasourceReader
	DashboardSrv  dashboardSearcher
	DatasourceMgr pluginManager
	AlertRuleSrv  alertRuleCreator
	CatalogSrv    catalogReader
}

// toolConstructor is a factory function that builds a single builtin tool from deps.
type toolConstructor func(deps BuiltinDeps) (tool.BaseTool, error)

// builtinRegistry holds all registered tool constructors.
// Each tool file adds itself via register() in its init() function.
var builtinRegistry []toolConstructor

// register appends a constructor to the package-level registry.
// Called from init() functions in each tool file.
func register(ctor toolConstructor) {
	builtinRegistry = append(builtinRegistry, ctor)
}
