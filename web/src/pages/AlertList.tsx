import {
  Card,
  Table,
  Tag,
  Space,
  Row,
  Col,
  Collapse,
  List,
  Checkbox,
  Typography,
  AvatarGroup,
  Avatar,
  TagInput,
  Button,
  SideSheet,
  Input,
  Pagination,
  Divider,
} from '@douyinfe/semi-ui';
import {
  IconMute,
  IconSearchStroked,
  IconHistory,
  IconSaveStroked,
  IconFilter,
  IconStarStroked,
} from '@douyinfe/semi-icons';
import { AlertSrv } from '@src/services';
import { useQuery } from '@tanstack/react-query';
import React, { useState } from 'react';
import { Chart } from '@src/components';
import { useNavigate } from '@src/hooks';

const { Text } = Typography;

const History: React.FC = () => {
  const [visible, setVisible] = useState(false);
  return (
    <>
      <Button icon={<IconHistory />} type="tertiary" onClick={() => setVisible(true)} />
      <SideSheet
        motion={false}
        closeOnEsc
        title="Alert filter history"
        visible={visible}
        onCancel={() => setVisible(false)}>
        <Input prefix={<IconFilter />} style={{ marginBottom: 12 }} />
        <List
          className="component-list-demo-booklist"
          dataSource={[
            {
              name: 'CPU Uage High',
              filter: 'Muted:true AND type:cpu',
            },
            {
              name: 'Disk Usage High',
              filter: 'Muted:true AND type:disk',
            },
            {
              name: 'Order Down',
              filter: 'Muted:true AND type:order',
            },
          ]}
          renderItem={(item: any) => (
            <List.Item
              className="list-item"
              onClick={() => {
                setVisible(false);
              }}
              style={{ padding: '6px 0px 6px 0px' }}
              header={<IconStarStroked size="large" style={{ paddingTop: 12 }} />}
              main={
                <div>
                  <span style={{ color: 'var(--semi-color-text-0)', fontWeight: 500 }}>{item.name}</span>
                  <p style={{ color: 'var(--semi-color-text-2)', margin: '4px 0', width: 500 }}>{item.filter}</p>
                </div>
              }
            />
          )}
        />
      </SideSheet>
    </>
  );
};

const AlertList: React.FC = () => {
  const navigate = useNavigate();
  const { data, isLoading } = useQuery(['alert-list'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return AlertSrv.getAlertList();
  });
  return (
    <Card style={{ padding: 0 }}>
      <div style={{ display: 'flex' }}>
        <TagInput
          prefix={<IconSearchStroked />}
          style={{ marginBottom: 24 }}
          defaultValue={['Muted:true', 'Priority:P1']}
        />
        <Button icon={<IconSaveStroked />} type="tertiary" style={{ marginLeft: 8, marginRight: 4 }} />
        <History />
      </div>
      <Row gutter={32} style={{ marginLeft: 0 }}>
        <Col span={4} style={{ outline: '1px solid var(--semi-color-border)' }}>
          <Collapse defaultActiveKey={['1', '2', '3', '4']}>
            <Collapse.Panel itemKey="1" header="Status">
              <List split={false}>
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      40
                    </Tag>
                  }
                  main={<Checkbox>Triggered</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      25
                    </Tag>
                  }
                  main={
                    <Checkbox>
                      <Tag
                        style={{ width: 4, backgroundColor: 'var(--semi-color-danger)', padding: 0, marginRight: 2 }}
                      />
                      Alert
                    </Checkbox>
                  }
                  style={{ marginLeft: 16, padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      12
                    </Tag>
                  }
                  main={
                    <Checkbox>
                      <Tag
                        style={{ width: 4, backgroundColor: 'var(--semi-color-warning)', padding: 0, marginRight: 2 }}
                      />
                      Warn
                    </Checkbox>
                  }
                  style={{ marginLeft: 16, padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      3
                    </Tag>
                  }
                  main={
                    <Checkbox>
                      <Tag
                        style={{ backgroundColor: 'var(--semi-color-text-3)', width: 4, padding: 0, marginRight: 2 }}
                      />
                      No Data
                    </Checkbox>
                  }
                  style={{ marginLeft: 16, padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      5
                    </Tag>
                  }
                  main={
                    <Checkbox>
                      <Tag
                        style={{ width: 4, backgroundColor: 'var(--semi-color-success)', padding: 0, marginRight: 2 }}
                      />
                      OK
                    </Checkbox>
                  }
                  style={{ padding: '4px 0px 4px 0px' }}
                />
              </List>
            </Collapse.Panel>
            <Collapse.Panel itemKey="2" header="Muted">
              <List split={false}>
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      20
                    </Tag>
                  }
                  main={<Checkbox checked>True</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      5
                    </Tag>
                  }
                  main={<Checkbox>False</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
              </List>
            </Collapse.Panel>
            <Collapse.Panel itemKey="3" header="Priority">
              <List split={false}>
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      1
                    </Tag>
                  }
                  main={<Checkbox>P1(Critical)</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      10
                    </Tag>
                  }
                  main={<Checkbox>P2(High)</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      3
                    </Tag>
                  }
                  main={<Checkbox>P3(Warn)</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      1
                    </Tag>
                  }
                  main={<Checkbox>Not Defined</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
              </List>
            </Collapse.Panel>
            <Collapse.Panel itemKey="4" header="Type">
              <List split={false}>
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      1
                    </Tag>
                  }
                  main={<Checkbox>Host</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      10
                    </Tag>
                  }
                  main={<Checkbox>APM</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      3
                    </Tag>
                  }
                  main={<Checkbox>Network</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
                <List.Item
                  extra={
                    <Tag size="small" shape="circle">
                      1
                    </Tag>
                  }
                  main={<Checkbox>RUM</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
              </List>
            </Collapse.Panel>
          </Collapse>
        </Col>
        <Col span={20}>
          <Chart type="bar" data={AlertSrv.getAlertStats()} height={180} />
          <Divider />
          <div style={{ display: 'flex', marginTop: 6, marginBottom: 6 }}>
            <Pagination showTotal total={20} size="small" style={{ width: '100%' }} />
            <Button type="tertiary" style={{ marginRight: 2 }}>
              Muted
            </Button>
            <Button type="tertiary">Resolve</Button>
          </div>
          <Divider />
          <Table
            rowKey="id"
            sticky
            size="small"
            rowSelection={{ fixed: 'left' }}
            loading={isLoading}
            dataSource={data}
            pagination={false}
            columns={[
              {
                title: 'STATUS',
                key: 'status',
                width: 100,
                align: 'center',
                dataIndex: 'status',
                render: (text: any) => {
                  return (
                    <Tag
                      type="solid"
                      style={{
                        width: 80,
                        backgroundColor: text === 'ALERT' ? 'var(--semi-color-danger)' : 'var(--semi-color-success)',
                      }}>
                      {text}
                    </Tag>
                  );
                },
              },
              {
                title: 'PRIORITY',
                key: 'priority',
                align: 'center',
                width: 100,
                dataIndex: 'priority',
              },
              {
                title: 'MUTED',
                width: 100,
                align: 'center',
                render: () => {
                  return (
                    <Space style={{ columnGap: 2 }} align="center">
                      <IconMute style={{ color: 'var(--semi-color-warning)' }} />
                      <Text type="tertiary">2</Text>
                      <Text size="small" type="tertiary">
                        hours
                      </Text>
                    </Space>
                  );
                },
              },
              {
                title: 'TITLE',
                key: 'name',
                dataIndex: 'name',
                render: (text: any) => {
                  return (
                    <Text link onClick={() => navigate('/alert/alert')}>
                      {text}
                    </Text>
                  );
                },
              },
              {
                title: 'TAGS',
                dataIndex: 'tags',
                render: (tags: any) => {
                  return (
                    <Space>
                      {(tags || []).map((t: string) => (
                        <Tag key={t}>{t}</Tag>
                      ))}
                    </Space>
                  );
                },
              },
              {
                title: 'RECEIVERS',
                align: 'right',
                render: () => {
                  return (
                    <Space>
                      <AvatarGroup overlapFrom={'start'} size="extra-small" maxCount={3}>
                        <Avatar color="red" alt="Lisa LeBlanc">
                          LL
                        </Avatar>
                        <Avatar alt="Caroline Xiao">CX</Avatar>
                        <Avatar color="amber" alt="Rafal Matin">
                          RM
                        </Avatar>
                        <Avatar style={{ color: '#f56a00', backgroundColor: '#fde3cf' }} alt="Zank Lance">
                          ZL
                        </Avatar>
                        <Avatar style={{ backgroundColor: '#87d068' }} alt="Youself Zhang">
                          YZ
                        </Avatar>
                      </AvatarGroup>
                    </Space>
                  );
                },
              },
            ]}
          />
        </Col>
      </Row>
    </Card>
  );
};

export default AlertList;
