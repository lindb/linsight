package schedule

import (
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/lindb/linsight/alerting/receivers/slack"
)

func Test(t *testing.T) {
	c := make(chan struct{})
	var wait sync.WaitGroup
	var wait2 sync.WaitGroup
	wait.Add(1)
	wait2.Add(1)
	go func() {
		c <- struct{}{}
	}()
	time.Sleep(time.Second)
	go func() {
		select {
		case <-c:
			fmt.Println("KKKK")
		default:
			fmt.Println("default")
		}
		wait.Done()
	}()
	wait.Wait()
}

// func TestRule(t *testing.T) {
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()
//
// 	datasourceSrv := service.NewMockDatasourceService(ctrl)
// 	datasourceSrv.EXPECT().GetDatasourceByUID(gomock.Any(), gomock.Any()).Return(&model.Datasource{
// 		Type:   model.LinDBDatasource,
// 		URL:    "http://localhost:9000",
// 		Config: datatypes.JSON(`{"database":"_internal"}`),
// 	}, nil).AnyTimes()
//
// 	querier := NewQuerier(&Deps{
// 		DatasourceSrv: datasourceSrv,
// 		DatasourceMgr: datasource.NewDatasourceManager(),
// 	})
// 	rs, err := querier.Query([]*model.Query{
// 		{
// 			Datasource: model.TargetDatasource{
// 				UID: "test",
// 			},
// 			Request: json.RawMessage(`{"namespace":"default-ns","metric":"lindb.monitor.system.mem_stat","fields":["usage"],"groupBy":["node","role"],"stats":false}`),
// 			RefID:   "A",
// 		},
// 	}, model.TimeRange{})
// 	var seriesList []*model.Series
// 	for refID, resultSet := range rs {
// 		for _, s := range resultSet.Series {
// 			for fn, f := range s.Fields {
// 				ts, val := getValue(f)
// 				seriesList = append(seriesList, &model.Series{
// 					RefID:      refID,
// 					MetricName: resultSet.MetricName,
// 					Field:      fn,
// 					Tags:       s.Tags,
// 					Timestamp:  ts,
// 					Value:      val,
// 				})
// 			}
// 		}
// 	}
// 	assert.NoError(t, err)
//
// 	r, err := NewAlertRule(&model.AlertRule{
// 		Conditions: datatypes.JSON(`[
// 			{
// 			"severity":1,
// 			"expr":"usage>100"
// 			},
// 			{
// 			"severity":2,
// 			"expr":"usage>10"
// 			}
// 		]`),
// 	})
// 	assert.NoError(t, err)
// 	alerts, err := r.Eval(context.TODO(), seriesList)
// 	assert.NoError(t, err)
// 	assert.Len(t, alerts, 2)
//
// 	slack, _ := types.GetReceiverPlugin("Slack")
// 	cli, err := slack.NewFn(json.RawMessage(`
// 		 {
//             "token": "xoxb-5107547294037-5885376189463-geaKuwUOja3stfKw8u9iK4Md"
//         }
// 		`), json.RawMessage(`{"channel":"#it"}`))
// 	assert.NoError(t, err)
// 	cli.Notify(context.TODO(), alerts)
// }
