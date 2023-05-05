import {
  Button,
  Typography,
  Card,
  Col,
  Row,
  Table,
  TagInput,
  Collapse,
  List,
  Checkbox,
  RadioGroup,
  Radio,
  SideSheet,
  Descriptions,
  Tag,
  Space,
  Divider,
} from '@douyinfe/semi-ui';
import {
  IconHistory,
  IconSearchStroked,
  IconPlusStroked,
  IconCopy,
  IconBell,
  IconStar,
  IconStarStroked,
  IconSearch,
} from '@douyinfe/semi-icons';
import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { AlertSrv, DashboardSrv } from '@src/services';
import { AddToDashboard, Chart } from '@src/components';
const { Text } = Typography;

const MetricsDetail: React.FC<{ metric: any; visible: boolean; setVisible: any }> = (props) => {
  const { metric, visible, setVisible } = props;
  if (!metric) {
    return null;
  }
  return (
    <SideSheet
      size="large"
      closeOnEsc
      visible={visible}
      motion={false}
      onCancel={() => setVisible(false)}
      mask={false}
      title={
        <Space>
          <span>{metric.value}</span>
          <IconCopy style={{ cursor: 'pointer' }} />
        </Space>
      }>
      <Collapse defaultActiveKey={['1', '2', '3', '4']} expandIconPosition="left">
        <Collapse.Panel itemKey="1" header="Metadata">
          <Descriptions
            data={[
              { key: 'Description', value: metric.name },
              { key: 'Metric Type', value: 'Gauge' },
              { key: 'Unit', value: 'Byte' },
              { key: 'Integration', value: <Text link>Kubernetes</Text> },
              {
                key: 'Tags',
                value: (
                  <Space>
                    <Tag>Env</Tag>
                    <Tag>Host</Tag>
                    <Tag>Team</Tag>
                    <Tag>Service</Tag>
                  </Space>
                ),
              },
            ]}
          />
        </Collapse.Panel>
        <Collapse.Panel itemKey="2" header="Preview Data">
          <div style={{ display: 'flex' }}>
            <div style={{ width: '100%' }}>
              <Button
                theme="borderless"
                icon={<IconSearch />}
                onClick={() => window.open(`${window.location.origin}/dashboard/explore`)}>
                Metric Explore
              </Button>
              <Divider layout="vertical" />
              <Button
                theme="borderless"
                icon={<IconBell />}
                onClick={() => window.open(`${window.location.origin}/alert/config`)}>
                Alert
              </Button>
              <Divider layout="vertical" />
              <AddToDashboard />
            </div>
            <Space style={{ width: 120 }}>
              <IconHistory />
              <span>Last 1 Hour</span>
            </Space>
          </div>
          <Divider style={{ marginTop: 8, marginBottom: 8 }} />
          <div>
            <Chart type="line" data={AlertSrv.getAlertStats()} height={230} />
          </div>
        </Collapse.Panel>
      </Collapse>
    </SideSheet>
  );
};

const Metrics: React.FC = () => {
  const [visible, setVisible] = useState(false);
  const [metric, setMetric] = useState(null);
  const { data, isLoading } = useQuery(['metrics-list'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return DashboardSrv.getMetricsList();
  });
  return (
    <>
      <Card bodyStyle={{ padding: 0 }}>
        <div style={{ margin: 16 }}>
          <TagInput prefix={<IconSearchStroked />} placeholder="Filter dashboard" defaultValue={['team:monitor']} />
        </div>
        <Row>
          <Col span={4}>
            <Collapse defaultActiveKey={['1', '2', '3', '4']}>
              <Collapse.Panel itemKey="1" header="Favorite">
                <RadioGroup direction="vertical" defaultValue="1">
                  <Radio value="1">Any</Radio>
                  <Radio value="2">Yes</Radio>
                  <Radio value="3">No</Radio>
                </RadioGroup>
              </Collapse.Panel>
              <Collapse.Panel itemKey="3" header="Metric List">
                <List split={false}>
                  <List.Item main={<Checkbox>Gauge</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Count</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Max</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Histogram</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                </List>
              </Collapse.Panel>
              <Collapse.Panel itemKey="3" header="Preset">
                <List split={false}>
                  <List.Item main={<Checkbox>All Custom</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>All Builtin</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                </List>
              </Collapse.Panel>
            </Collapse>
          </Col>
          <Col span={20}>
            <div style={{ marginLeft: 16, marginRight: 16, marginBottom: 16 }}>
              <Card
                bodyStyle={{ padding: 0 }}
                title={<Card.Meta title="Metrics" description="3.2k total" />}
                headerExtraContent={
                  <>
                    <Button icon={<IconPlusStroked />}>New</Button>
                  </>
                }>
                <Table
                  rowKey="id"
                  showHeader
                  className="x-monitor"
                  size="small"
                  dataSource={data}
                  loading={isLoading}
                  pagination={{ size: 'small', style: { marginLeft: 8 }, pageSize: 20 }}
                  columns={[
                    {
                      title: <IconStar />,
                      key: 'favorite',
                      width: 40,
                      align: 'center',
                      dataIndex: 'status',
                      render: (_text: any, r: any, _index: any) => {
                        if (r.favorite) {
                          return <IconStar style={{ cursor: 'pointer' }} />;
                        }
                        return <IconStarStroked style={{ cursor: 'pointer' }} />;
                      },
                    },
                    {
                      title: 'Description',
                      key: 'name',
                      align: 'left',
                      dataIndex: 'name',
                      render: (_text: any, r: any, _index: any) => {
                        return (
                          <>
                            {r.type && <i style={{ marginRight: 4 }} className={`devicon-${r.type}-plain colored`} />}
                            <Text
                              link
                              onClick={() => {
                                setMetric(r);
                                setVisible(true);
                              }}>
                              {r.name}
                            </Text>
                          </>
                        );
                      },
                    },
                    {
                      title: 'Name',
                      key: 'value',
                      align: 'left',
                      dataIndex: 'value',
                      render: (text: any, r: any) => {
                        return (
                          <Text
                            link
                            onClick={() => {
                              setMetric(r);
                              setVisible(true);
                            }}>
                            {text}
                          </Text>
                        );
                      },
                    },
                  ]}
                />
              </Card>
            </div>
          </Col>
        </Row>
      </Card>
      <MetricsDetail metric={metric} visible={visible} setVisible={setVisible} />
    </>
  );
};

export default Metrics;
