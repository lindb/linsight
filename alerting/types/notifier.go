package types

import (
	"context"
	"encoding/json"

	"github.com/lindb/linsight/alerting/template"
)

type NewNotifier func(setting, cfg json.RawMessage) (Notifier, error)

// Notifier sends alert notifications through a specific channel (e.g. Slack, Email).
type Notifier interface {
	Notify(ctx context.Context, data *template.NotificationData) error
}
