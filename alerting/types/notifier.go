package types

import (
	"context"
	"encoding/json"

	"github.com/lindb/linsight/model"
)

type NewNotifier func(setting, cfg json.RawMessage) (Notifier, error)

type Notifier interface {
	Notify(ctx context.Context, alerts []*model.Alert) error
}
