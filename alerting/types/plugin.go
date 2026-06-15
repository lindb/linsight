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

package types

import (
	"github.com/lindb/linsight/model"
)

// NotifierDef is the runtime definition of a notifier plugin.
// Plugin holds the frontend metadata (JSON schemas); NewFn is the factory for creating notifier instances.
// SummaryKey is the config JSON field whose value is surfaced in the contact-point list (e.g. "addresses", "channel", "url").
type NotifierDef struct {
	Plugin     *model.NotifierPlugin
	NewFn      NewNotifier
	SummaryKey string // config JSON key shown as primary info in the contact-point list
}

var (
	notifierDefs = make(map[string]*NotifierDef)
)

// Register adds a notifier plugin definition to the global registry.
func Register(def *NotifierDef) {
	notifierDefs[def.Plugin.Type] = def
}

// GetNotifierPlugins returns the frontend metadata for all registered notifier plugins.
func GetNotifierPlugins() (rs []*model.NotifierPlugin) {
	for _, d := range notifierDefs {
		rs = append(rs, d.Plugin)
	}
	return rs
}

// GetNotifierDef looks up a notifier plugin definition by its type string.
func GetNotifierDef(pluginType string) (*NotifierDef, bool) {
	def, ok := notifierDefs[pluginType]
	return def, ok
}
