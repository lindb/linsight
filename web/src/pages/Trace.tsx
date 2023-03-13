import { Card, Space, Typography, Tag, Dropdown } from '@douyinfe/semi-ui';
import { APMSrv } from '@src/services';
import { useQuery } from '@tanstack/react-query';
import {
  IconDescend,
  IconGridStroked,
  IconInheritStroked,
  IconFlowChartStroked,
  IconPieChart2Stroked,
} from '@douyinfe/semi-icons';
import React, { useState } from 'react';
import * as _ from 'lodash-es';
import moment from 'moment';
import { TraceFlame, TraceTimeline, TraceMap, TraceSummary, TraceFlame2 } from '@src/components';

const { Text, Title } = Typography;

const Trace: React.FC = () => {
  const { data, isLoading } = useQuery(['trace-data'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return APMSrv.getTraceData();
  });
  const [type, setType] = useState('map');
  const processes = _.get(data, 'processes', {});
  const renderRoot = () => {
    const root: any = _.get(data, 'spans[0]', null);
    if (!root) {
      return null;
    }
    const process: any = _.get(processes, root.processID, {});
    const language = _.get(
      _.find(
        _.get(process, 'tags', []),
        (o: any) => o.key === 'telemetry.sdk.language' || o.key === 'library.language'
      ),
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
                  <Title heading={4} style={{ marginLeft: 4, marginTop: 6 }} type="tertiary">
                    <span>{root.operationName}</span>
                  </Title>
                  <Tag type="solid" size="small" style={{ marginTop: 4, backgroundColor: 'var(--semi-color-success)' }}>
                    OK
                  </Tag>
                </Space>
              </Title>
            </>
          }
          description={
            <Space>
              <Text strong>Start Time: </Text>
              <Text type="tertiary">{moment(root.startTime / 1000).format('YYYY-MM-DD HH:mm:ss.SSS')}</Text>
              <Text strong>Duration: </Text>
              <Text type="tertiary">{root.duration / 1000} ms</Text>
              <Text strong>Trace ID: </Text>
              <Text type="tertiary" copyable>
                {root.traceID}
              </Text>
            </Space>
          }
        />
      </div>
    );
  };

  const renderContent = () => {
    const spans = _.get(data, 'spans', []);
    switch (type) {
      case 'timeline':
        return <TraceTimeline loading={isLoading} spans={spans} processes={processes} />;
      case 'flame':
        return <TraceFlame spans={spans} />;
      case 'flame2':
        return <TraceFlame2 />;
      case 'summary':
        return <TraceSummary />;
      case 'map':
        return <TraceMap spans={spans} processes={processes} />;
    }
  };
  return (
    <Card
      bodyStyle={{ padding: 8 }}
      title={renderRoot()}
      headerExtraContent={
        <Dropdown
          render={
            <Dropdown.Menu>
              <Dropdown.Item onClick={() => setType('timeline')} icon={<IconInheritStroked />}>
                Timeline
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setType('flame')} icon={<IconDescend />}>
                Flame Graph
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setType('flame2')} icon={<IconDescend />}>
                Flame Graph 2
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setType('map')} icon={<IconFlowChartStroked />}>
                Map
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setType('summary')} icon={<IconPieChart2Stroked />}>
                Summary
              </Dropdown.Item>
            </Dropdown.Menu>
          }>
          <IconGridStroked style={{ cursor: 'pointer' }} />
        </Dropdown>
      }>
      {renderContent()}
    </Card>
  );
};

export default Trace;
