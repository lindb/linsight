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
import { Card, Space, Table, Typography } from '@douyinfe/semi-ui';
import { IntegrationIcon } from '@src/components';
import { FormatRepositoryInst, Span, Trace } from '@src/types';
import moment from 'moment';
import React, { useEffect, useState } from 'react';
import SpanView from '../components/SpanView';
import { default as TraceKit } from '../util/trace';

const { Text } = Typography;

const TimelineView: React.FC<{ traces: Trace[] }> = (props) => {
  const { traces } = props;
  const [traceTree, setTraceTree] = useState<Span[]>([]);
  const [currentSpan, setCurrentSpan] = useState<Span | undefined>(undefined);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    setTraceTree(TraceKit.buildTraceTree(traces));
  }, [traces]);

  return (
    <Card className="linsight-feature timeline-view" bodyStyle={{ padding: 8 }}>
      {visible && <SpanView visible={visible} setVisible={setVisible} span={currentSpan} />}
      <Table
        bordered
        expandAllRows
        pagination={false}
        indentSize={15}
        size="small"
        dataSource={traceTree}
        rowKey="spanId"
        columns={[
          {
            title: 'Service & Operation',
            dataIndex: 'name',
            render: (_text: any, r: Span) => {
              const language = r.process.sdkLanguage;
              const serviceName = r.process.serviceName;
              return (
                <Space>
                  <div
                    style={{ display: 'flex', alignItems: 'center', gap: 4, cursor: 'pointer' }}
                    onClick={() => {
                      setCurrentSpan(r);
                      setVisible(true);
                    }}>
                    <Text strong icon={<IntegrationIcon integration={language} />}>
                      {serviceName}
                    </Text>
                    <Text type="tertiary">{r.name}</Text>
                  </div>
                </Space>
              );
            },
          },
          {
            title: 'Exec Time',
            dataIndex: 'duration',
            width: 150,
            render: (text: any, _r: Span) => {
              return FormatRepositoryInst.get('ns').formatString(text, 3);
            },
          },
          {
            title: 'Exec Time(%)',
            dataIndex: 'duration',
            key: 'duration%',
            width: 150,
            render: (_text: any, r: Span) => {
              const offset = r.startTime - r.traceStart;
              const percent = (r.total * 100) / r.traceTotal;
              return (
                <div className="duration-percent">
                  <div
                    className="inner"
                    style={{
                      backgroundColor: `var(--semi-color-${
                        percent > 80 ? 'danger' : percent > 50 ? 'warning' : 'success'
                      })`,
                      width: `${percent}%`,
                      marginLeft: `calc(${(offset * 100) / r.traceTotal}%)`,
                    }}></div>
                  <div className="percent-text">
                    <Text type="tertiary" size="small">
                      {percent.toFixed(2)}%
                    </Text>
                  </div>
                </div>
              );
            },
          },
          {
            title: 'Start Time',
            dataIndex: 'startTime',
            width: 120,
            render: (text: any) => {
              return moment(text / 1000).format('HH:mm:ss.SSS');
            },
          },
        ]}
      />
    </Card>
  );
};

export default TimelineView;
