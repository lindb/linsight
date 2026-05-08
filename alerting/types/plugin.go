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

type ReceiverPluginDefine struct {
	Define *model.ReceiverPlugin
	NewFn  NewNotifier
}

var (
	receiverMap = make(map[string]*ReceiverPluginDefine)
)

func Register(receiver *ReceiverPluginDefine) {
	receiverMap[receiver.Define.Type] = receiver
}

func GetReceiverPlugins() (rs []*model.ReceiverPlugin) {
	for _, rp := range receiverMap {
		rs = append(rs, rp.Define)
	}
	return rs
}

func GetReceiverPlugin(pluginType string) (*ReceiverPluginDefine, bool) {
	plugin, ok := receiverMap[pluginType]
	return plugin, ok
}
