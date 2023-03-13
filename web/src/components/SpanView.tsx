import { Card, Typography, SideSheet, Space, Tag, Tabs, TabPane, Descriptions, List, Divider } from '@douyinfe/semi-ui';
import React from 'react';
import * as _ from 'lodash-es';
import moment from 'moment';

const { Text, Title } = Typography;

const SpanView: React.FC<{ visible: boolean; setVisible: any; span: any; process: any; processes: any }> = (props) => {
  const { visible, setVisible, span, process, processes } = props;
  const logs = _.get(span, 'logs', []);
  const renderTitle = () => {
    if (!span) {
      return null;
    }

    const language = _.get(
      _.find(
        _.get(process, 'tags', []),
        (o: any) => o.key === 'telemetry.sdk.language' || o.key === 'library.language'
      ),
      'value',
      ''
    );
    const status = _.get(
      _.find(_.get(span, 'tags', []), (o: any) => o.key === 'otel.status_code'),
      'value',
      ''
    );
    return (
      <div>
        <Card.Meta
          title={
            <>
              <Title heading={2}>
                <Space>
                  {language && <i style={{ marginRight: 4 }} className={`devicon-${language}-plain colored`} />}
                  <Title heading={2}>{process.serviceName}</Title>
                  <Title
                    heading={4}
                    ellipsis={{ showTooltip: true }}
                    style={{ marginLeft: 4, marginTop: 2, width: 500 }}
                    type="tertiary">
                    {span.operationName}
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
              <Text type="tertiary">{span.spanID}</Text>
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
      title={renderTitle()}
      size="large"
      motion={false}
      mask={false}
      visible={visible}
      onCancel={() => setVisible(false)}>
      <Tabs type="line" size="small">
        <TabPane tab="Process" itemKey="1">
          <Descriptions data={_.get(process, 'tags', [])} />
        </TabPane>
        <TabPane tab="Tags" itemKey="2">
          <Descriptions data={_.get(span, 'tags', [])} />
        </TabPane>
        <TabPane
          tab={
            <>
              Logs <Tag>{logs.length}</Tag>
            </>
          }
          itemKey="3">
          <List
            bordered
            size="small"
            dataSource={logs}
            renderItem={(item: any) => (
              <List.Item>
                <Space>
                  {[
                    <span key="ttt-timestamp">{moment(item.timestamp / 1000).format('YYYY-MM-DD HH:mm:ss.SSS')}</span>,
                    ...item.fields.map((o: any, idx: number) => {
                      if (idx < item.fields.length) {
                        return (
                          <>
                            <Divider layout="vertical" />
                            <Text strong>{o.key}:</Text>
                            <Text>{o.value}</Text>
                          </>
                        );
                      }
                      return (
                        <>
                          <Text strong>{o.key}:</Text>
                          <Text>{o.value}</Text>
                        </>
                      );
                    }),
                  ]}
                </Space>
              </List.Item>
            )}
          />
        </TabPane>
        <TabPane tab="Service Metrics" itemKey="4"></TabPane>
        <TabPane tab="JVM/Go Runtime" itemKey="5"></TabPane>
        <TabPane tab="Infrastructure" itemKey="6"></TabPane>
      </Tabs>
    </SideSheet>
  );
};

export default SpanView;
