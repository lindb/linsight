import {
  Button,
  Card,
  Checkbox,
  CheckboxGroup,
  Dropdown,
  Input,
  Nav,
  Progress,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { APMSrv } from '@src/services';
import {
  IconGlobeStroke,
  IconElementStroked,
  IconSettingStroked,
  IconServerStroked,
  IconSourceControl,
  IconSearchStroked,
  IconStar,
  IconStarStroked,
} from '@douyinfe/semi-icons';
import { useQuery } from '@tanstack/react-query';
import React from 'react';
import * as _ from 'lodash-es';
import { useNavigate } from '@src/hooks';

const { Text, Numeral } = Typography;

const Service: React.FC = () => {
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(['servic_overview'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return APMSrv.getServiceStats();
  });
  return (
    <Card bodyStyle={{ padding: 0 }}>
      <Nav
        mode="horizontal"
        footer={
          <>
            <Input suffix={<IconSearchStroked />} style={{ marginRight: 12, width: 300 }} />
            <Dropdown
              trigger="click"
              render={
                <div>
                  <CheckboxGroup
                    style={{ width: 180, padding: 12 }}
                    defaultValue={['A', 'B']}
                    aria-label="CheckboxGroup 示例">
                    <Checkbox value="A">TYPE</Checkbox>
                    <Checkbox value="B">SERVICE</Checkbox>
                    <Checkbox value="C">REQUESTS</Checkbox>
                    <Checkbox value="D">AVG</Checkbox>
                    <Checkbox value="E">P95</Checkbox>
                    <Checkbox value="F">P99</Checkbox>
                  </CheckboxGroup>
                </div>
              }>
              <IconSettingStroked />
            </Dropdown>
          </>
        }>
        <Button icon={<IconGlobeStroke />} type="tertiary" theme="light">
          Web
        </Button>
        <Button icon={<IconElementStroked />} type="tertiary" theme="light">
          Service
        </Button>
        <Button icon={<IconServerStroked />} type="tertiary" theme="light">
          Database
        </Button>
        <Button icon={<IconSourceControl />} type="tertiary" theme="light">
          Queue
        </Button>
      </Nav>
      <Table
        size="small"
        dataSource={data}
        loading={isLoading}
        columns={[
          {
            key: 'f',
            title: <IconStar />,
            width: 20,
            render: (_text: any, _record: any, _index: any) => {
              return <IconStarStroked />;
            },
          },
          {
            key: 'type',
            title: 'TYPE',
            width: 70,
            align: 'center',
            render: (_text: any, record: any, _index: any) => {
              const type = _.get(record, 'tags.type');
              switch (type) {
                case 'php':
                  return <IconGlobeStroke />;
                case 'mysql':
                  return <IconServerStroked />;
                case 'apachekafka':
                  return <IconSourceControl />;
                default:
                  return <IconElementStroked />;
              }
            },
          },
          {
            key: 'service',
            title: 'SERVICE',
            align: 'left',
            render: (_text: any, record: any, _index: any) => {
              return (
                <Space align="center">
                  <i className={`devicon-${_.get(record, 'tags.type')}-plain colored`} />
                  <div>{_.get(record, 'tags.name', 'unknown')}</div>
                </Space>
              );
            },
          },
          {
            key: 'requests',
            width: 220,
            align: 'right',
            title: 'REQUESTS',
            render: (_text: any, record: any, _index: any) => {
              return (
                <div style={{ display: 'flex' }}>
                  <div style={{ width: 80, textAlign: 'right', marginRight: 4 }}>
                    <Text strong>{_.get(record, 'fields.requests', 20)}</Text>
                    <Text size="small" type="tertiary">
                      {' '}
                      req/s
                    </Text>
                  </div>
                  <Progress
                    style={{ width: 100, height: 14 }}
                    size="large"
                    percent={Number(_.get(record, 'fields.errorRate', 20))}
                  />
                </div>
              );
            },
          },
          {
            key: 'avg',
            width: 220,
            align: 'right',
            title: 'AVG.LATENCY',
            render: (_text: any, record: any, _index: any) => {
              return (
                <div style={{ display: 'flex' }}>
                  <div style={{ width: 80, textAlign: 'right', marginRight: 4 }}>
                    <Text strong>{_.get(record, 'fields.avg', 20)}</Text>
                    <Text size="small" type="tertiary">
                      {' '}
                      ms
                    </Text>
                  </div>
                  <Progress
                    style={{ width: 100, height: 14 }}
                    size="large"
                    percent={Number(_.get(record, 'fields.errorRate', 20))}
                  />
                </div>
              );
            },
          },
          {
            key: 'p95',
            width: 220,
            align: 'right',
            title: 'P95 LATENCY',
            render: (_text: any, record: any, _index: any) => {
              return (
                <div style={{ display: 'flex' }}>
                  <div style={{ width: 80, textAlign: 'right', marginRight: 4 }}>
                    <Text strong>{_.get(record, 'fields.p95', 20)}</Text>
                    <Text size="small" type="tertiary">
                      {' '}
                      ms
                    </Text>
                  </div>
                  <Progress
                    style={{ width: 100, height: 14 }}
                    size="large"
                    percent={Number(_.get(record, 'fields.errorRate', 20))}
                  />
                </div>
              );
            },
          },
          {
            key: 'error',
            width: 220,
            align: 'right',
            title: 'ERROR RATE',
            render: (_text: any, record: any, _index: any) => {
              return (
                <div style={{ display: 'flex' }}>
                  <div style={{ width: 80, textAlign: 'right', marginRight: 4 }}>
                    <Numeral strong rule="percentages">
                      {_.get(record, 'fields.errorRate', 20)}
                    </Numeral>
                  </div>
                  <Progress
                    style={{ width: 100, height: 14 }}
                    size="large"
                    percent={Number(_.get(record, 'fields.errorRate', 20))}
                  />
                </div>
              );
            },
          },
          {
            key: 'MONITOR STATUS',
            title: 'MONITOR STATUS',
            align: 'center',
            width: 160,
            render: (_text: any, record: any, _index: any) => {
              const alert = _.get(record, 'fields.alert', 20);
              return (
                <Tag
                  type="solid"
                  onClick={() => navigate('/alert/alert-list')}
                  style={{
                    width: 100,
                    cursor: 'pointer',
                    backgroundColor: alert > 0 ? 'var(--semi-color-danger)' : 'var(--semi-color-success)',
                  }}>
                  {`${alert} ${alert > 0 ? 'ALERT' : 'OK'}`}
                </Tag>
              );
            },
          },
        ]}
        pagination={false}
      />
    </Card>
  );
};

export default Service;
