/*
Licensed to LinDB under one or more contributor
license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright
ownership. LinDB licenses this file to you under
the Apache License, Version 2.0 (the "License"); you may
not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
 
Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/
import { List, SideSheet, Typography } from '@douyinfe/semi-ui';
import { useRequest } from '@src/hooks';
import { ChartSrv } from '@src/services';
import { Dashboard } from '@src/types';
import { isEmpty } from 'lodash-es';
import React from 'react';

const { Text } = Typography;

const ListDashboardByChart: React.FC<{
  chartUid: string;
  visible: boolean;
  setVisible: (visible: boolean) => void;
}> = (props) => {
  const { chartUid, visible, setVisible } = props;
  const { result, loading } = useRequest(
    ['load_dashboards_by_chart', chartUid],
    async () => {
      return ChartSrv.getDashboardsByChart(chartUid);
    },
    { enabled: !isEmpty(chartUid) }
  );
  return (
    <SideSheet
      className="chart-detail"
      closeOnEsc
      title="The following dashboards using this chart"
      motion={false}
      closable
      visible={visible}
      onCancel={() => setVisible(false)}>
      <List
        header="Dashboard"
        loading={loading}
        bordered
        dataSource={result}
        renderItem={(item: Dashboard) => {
          return (
            <List.Item key={item.uid}>
              <Text>{item.title}</Text>
            </List.Item>
          );
        }}
      />
    </SideSheet>
  );
};

export default ListDashboardByChart;
