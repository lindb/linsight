package mail

import (
	_ "embed"
	"encoding/json"

	"github.com/lindb/linsight/alerting/types"
	"github.com/lindb/linsight/model"
)

//go:embed setting.json
var setting string

func init() {
	types.Register(&types.ReceiverPluginDefine{
		Define: &model.ReceiverPlugin{
			Type:    "Email",
			Setting: json.RawMessage(setting),
		},
	})
}
