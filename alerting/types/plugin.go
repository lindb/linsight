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
