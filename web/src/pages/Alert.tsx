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
import React from 'react';
import {
  Avatar,
  Button,
  Card,
  Checkbox,
  Collapse,
  Divider,
  Dropdown,
  List,
  Space,
  Tag,
  TagInput,
  Typography,
  Descriptions,
} from '@douyinfe/semi-ui';
import { IconMute, IconActivity, IconSettingStroked, IconReplyStroked, IconSearchStroked } from '@douyinfe/semi-icons';
import { useQuery } from '@tanstack/react-query';
import { AlertSrv } from '@src/services';
import { useNavigate } from '@src/hooks';

const { Text, Title } = Typography;

const AlertEvents: React.FC = () => {
  const { data, isLoading } = useQuery(['alert-events'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return AlertSrv.getAlertEvents();
  });
  return (
    <List
      loading={isLoading}
      dataSource={data}
      renderItem={(item) => (
        <List.Item
          style={{ paddingLeft: 0, cursor: 'pointer' }}
          header={<Avatar color="light-blue">XM</Avatar>}
          main={
            <div>
              <span style={{ color: 'var(--semi-color-text-2)' }}>{item.date}</span>
              <div>
                <Tag type="solid" style={{ backgroundColor: `var(--semi-color-${item.color})` }}>
                  {item.status}
                </Tag>
                <span style={{ color: 'var(--semi-color-text-0)', fontWeight: 500, marginLeft: 6 }}>{item.title}</span>
              </div>
              <div style={{ color: 'var(--semi-color-text-0)', fontWeight: 500, marginTop: 6, marginBottom: 6 }}>
                {item.notify}
              </div>
              <div style={{ color: 'var(--semi-color-text-2)', margin: '4px 0' }}>
                <img src="/alert.png" />
              </div>
              <div style={{ color: 'var(--semi-color-text-0)', fontWeight: 500 }}>{item.content}</div>
              <div style={{ color: 'var(--semi-color-text-0)' }}>{item.last}</div>
            </div>
          }
        />
      )}
    />
  );
};

const Alert: React.FC = () => {
  const navigate = useNavigate();
  const data = [
    {
      key: 'Query',
      value: <Tag style={{ margin: 0 }}>{`max(last_5m):avg:system.cpu.user{host:192.168.0.1} > 5`}</Tag>,
    },
    { key: 'Message', value: `{{#is_alert}} your host cpu is high than 5%, now is {{value}} {{/is_alert}}` },
    { key: 'Priority', value: 'P1(Critical)' },
    {
      key: 'Tags',
      value: (
        <Space>
          <Tag>env:prod</Tag>
          <Tag>team:monitor</Tag>
        </Space>
      ),
    },
  ];
  return (
    <div>
      <Card
        headerExtraContent={
          <div>
            <Button icon={<IconReplyStroked />} onClick={() => navigate('/alert/alert-list')} />
            <Button icon={<IconMute />} type="tertiary" style={{ marginLeft: 4, marginRight: 4 }}>
              Muted
            </Button>
            <Dropdown
              render={
                <Dropdown.Menu>
                  <Dropdown.Item>Edit</Dropdown.Item>
                  <Dropdown.Item>Clone</Dropdown.Item>
                  <Dropdown.Item>
                    <Text type="danger">Delete</Text>
                  </Dropdown.Item>
                </Dropdown.Menu>
              }>
              <Button icon={<IconSettingStroked />} type="tertiary" />
            </Dropdown>
          </div>
        }
        title={
          <div>
            <Title heading={4} type="success">{`CPU Load is high on {{host.name}}`}</Title>
            <div>
              <Tag style={{ marginRight: 8, backgroundColor: 'var(--semi-color-success)' }}>OK</Tag>
              <span>Monitor status since 20 hours and 14 minutes ago</span>
            </div>
          </div>
        }
        bodyStyle={{ padding: 0 }}>
        <Collapse expandIconPosition="left" accordion defaultActiveKey={['1']}>
          <Collapse.Panel header="Properties" itemKey="1">
            <Descriptions data={data} />
          </Collapse.Panel>
        </Collapse>
      </Card>
      <Card style={{ marginTop: 12 }} bodyStyle={{ padding: 0 }}>
        <div style={{ margin: 16 }}>
          <TagInput prefix={<IconSearchStroked />} placeholder="Filter alert type and their events" />
        </div>
        <Space style={{ marginLeft: 16, marginBottom: 16, display: 'flex' }}>
          <Space style={{ display: 'inline-flex', width: '100%' }}>
            <Checkbox>
              <Tag style={{ width: 4, backgroundColor: 'var(--semi-color-danger)', padding: 0, marginRight: 2 }} />
              Alert
            </Checkbox>
            <Checkbox>
              <Tag style={{ width: 4, backgroundColor: 'var(--semi-color-warning)', padding: 0, marginRight: 2 }} />
              Warn
            </Checkbox>
            <Checkbox>
              <Tag style={{ width: 4, backgroundColor: 'var(--semi-color-text-3)', padding: 0, marginRight: 2 }} />
              No data
            </Checkbox>
            <Checkbox>
              <Tag style={{ width: 4, backgroundColor: 'var(--semi-color-success)', padding: 0, marginRight: 2 }} />
              OK
            </Checkbox>
          </Space>
          <div style={{ marginRight: 16 }}>
            <Button icon={<IconActivity />} type="tertiary">
              Last 1 Hour
            </Button>
          </div>
        </Space>
        <Divider />
        <Collapse expandIconPosition="left" defaultActiveKey={['1', '2']}>
          <Collapse.Panel header="Status & History" itemKey="1"></Collapse.Panel>
          <Collapse.Panel header="Events" itemKey="2">
            <AlertEvents />
          </Collapse.Panel>
        </Collapse>
      </Card>
    </div>
  );
};

export default Alert;
