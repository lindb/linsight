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
import { Card, Typography, SideSheet, Space, Tag, Tabs, TabPane, Descriptions, List, Divider } from '@douyinfe/semi-ui';
import React from 'react';
import moment from 'moment';
import { Span } from '@src/types';
import { IntegrationIcon } from '@src/components';
import { get } from 'lodash-es';
import { default as TraceKit } from '../util/trace';

const { Text, Title } = Typography;

const SpanView: React.FC<{ visible: boolean; setVisible: (v: boolean) => void; span?: Span }> = (props) => {
  const { visible, setVisible, span } = props;
  if (!span) {
    return null;
  }
  const events = get(span, 'events', []);
  const renderTitle = () => {
    const status = 'ok';

    return (
      <div>
        <Card.Meta
          title={
            <>
              <Title heading={2}>
                <Space>
                  <IntegrationIcon integration={span.process.sdkLanguage} />
                  <Title heading={2}>{span.process.serviceName}</Title>
                  <Title
                    heading={4}
                    ellipsis={{ showTooltip: true }}
                    style={{ marginLeft: 4, marginTop: 2, width: 500 }}
                    type="tertiary">
                    {span.name}
                  </Title>
                </Space>
              </Title>
            </>
          }
          description={
            <Space>
              <Text strong>Start Time: </Text>
              <Text type="tertiary">{moment(span.startTime / 1000).format('YYYY-MM-DD HH:mm:ss.SSS')}</Text>
              <Text strong>Duration: </Text>
              <Text type="tertiary">{span.duration / 1000} ms</Text>
              <Text strong>Span ID: </Text>
              <Text type="tertiary">{span.spanId}</Text>
              <Tag
                type="solid"
                size="small"
                style={{ backgroundColor: `var(--semi-color-${status === 'ERROR' ? 'danger' : 'success'})` }}>
                {status === 'ERROR' ? 'ERROR' : 'OK'}
              </Tag>
            </Space>
          }
        />
      </div>
    );
  };

  return (
    <SideSheet
      closeOnEsc
      closable={false}
      title={renderTitle()}
      size="large"
      motion={false}
      mask={false}
      visible={visible}
      onCancel={() => setVisible(false)}>
      <Tabs type="line" size="small">
        <TabPane tab="Process" itemKey="1">
          <Descriptions data={TraceKit.toKeyValueList(get(span, 'process.tags', {}))} />
        </TabPane>
        <TabPane tab="Tags" itemKey="2">
          <Descriptions data={TraceKit.toKeyValueList(get(span, 'tags', {}))} />
        </TabPane>
        <TabPane
          tab={
            <>
              Events <Tag>{events.length}</Tag>
            </>
          }
          itemKey="3">
          <List
            bordered
            size="small"
            dataSource={events}
            renderItem={(item: any) => (
              <List.Item>
                <Space>
                  {[
                    <span key="ttt-timestamp">
                      {moment(item.timestamp / 1000000).format('YYYY-MM-DD HH:mm:ss.SSS')}
                    </span>,
                  ]}
                </Space>
              </List.Item>
            )}
          />
        </TabPane>
      </Tabs>
    </SideSheet>
  );
};

export default SpanView;
