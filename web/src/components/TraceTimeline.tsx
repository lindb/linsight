import { Table, Typography, Space } from '@douyinfe/semi-ui';
import { IconAlertTriangle, IconBriefStroked } from '@douyinfe/semi-icons';
import React, { useState } from 'react';
import * as _ from 'lodash-es';
import moment from 'moment';
import SpanView from './SpanView';

const { Text } = Typography;

const TraceTimeline: React.FC<{ loading: boolean; spans: any; processes: any }> = (props) => {
  const { processes, spans, loading } = props;
  const [currentSpan, setCurrentSpan] = useState(null);
  const [currentProcess, setCurrentProcess] = useState(null);
  const [visible, setVisible] = useState(false);
  return (
    <>
      <SpanView
        visible={visible}
        setVisible={setVisible}
        span={currentSpan}
        process={currentProcess}
        processes={processes}
      />
      <Table
        className="x-monitor"
        size="small"
        pagination={false}
        dataSource={spans}
        rowKey="spanID"
        expandAllRows
        loading={loading}
        indentSize={10}
        columns={[
          {
            title: 'Service & Operation',
            fixed: true,
            dataIndex: 'operationName',
            render: (text: any, r: any) => {
              const process: any = _.get(processes, r.processID, {});
              const language = _.get(
                _.find(
                  _.get(process, 'tags', []),
                  (o: any) => o.key === 'telemetry.sdk.language' || o.key === 'library.language'
                ),
                'value',
                ''
              );
              const status = _.get(
                _.find(_.get(r, 'tags', []), (o: any) => o.key === 'otel.status_code'),
                'value',
                ''
              );
              return (
                <Space>
                  <Text
                    strong
                    onClick={() => {
                      setCurrentProcess(process);
                      setCurrentSpan(r);
                      setVisible(true);
                    }}
                    style={{ cursor: 'pointer' }}
                    icon={language && <i style={{ marginRight: 4 }} className={`devicon-${language}-plain colored`} />}>
                    {process.serviceName}
                  </Text>
                  <Text style={{ marginLeft: 4, cursor: 'pointer' }} type="tertiary">
                    {text}
                  </Text>
                  {!_.isEmpty(_.get(r, 'logs', [])) && <IconBriefStroked />}
                  {status === 'ERROR' && <IconAlertTriangle style={{ color: 'var(--semi-color-danger)' }} />}
                </Space>
              );
            },
          },
          {
            title: 'Duration',
            dataIndex: 'duration',
            className: 'duration',
            width: 100,
            align: 'right',
            render: (text: any) => {
              return (
                <div>
                  <Text>{text / 100}</Text>
                  <Text size="small" type="tertiary">
                    {' '}
                    ms
                  </Text>
                </div>
              );
            },
          },
          {
            title: 'Duration(%)',
            dataIndex: 'duration',
            key: 'duration%',
            width: 150,
            render: (text: any, r: any) => {
              const total = 24476;
              const offset = r.startTime - 1669255471995014;
              const percent = (text * 100) / total;
              return (
                <div className="timeline">
                  <div
                    className="inner"
                    style={{
                      backgroundColor: `var(--semi-color-${
                        percent > 80 ? 'danger' : percent > 50 ? 'warning' : 'success'
                      })`,
                      width: `${percent}%`,
                      marginLeft: `calc(${(offset * 100) / total}%)`,
                    }}></div>
                  <div style={{ position: 'absolute', top: 0, fontSize: 12, textAlign: 'center', width: '100%' }}>
                    <Text type="tertiary">{percent.toFixed(2)}%</Text>
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
    </>
  );
};
export default TraceTimeline;
