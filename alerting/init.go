package alerting

import (
	_ "github.com/lindb/linsight/alerting/receivers/email"
	_ "github.com/lindb/linsight/alerting/receivers/slack"
	_ "github.com/lindb/linsight/alerting/receivers/webhook"
)
