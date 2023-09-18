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
import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import { DynamicForm, Icon } from '@src/components';
import { Collapse, Card, Typography, Button, Form, Select } from '@douyinfe/semi-ui';
import {
  IconCopy,
  IconDeleteStroked,
  IconPlay,
  IconSaveStroked,
  IconPlusStroked,
  IconChevronDown,
} from '@douyinfe/semi-icons';
import { cloneDeep, find, get, isEmpty } from 'lodash-es';
import { PlatformContext } from '@src/contexts';
import { ReceiverConfig } from '@src/types';
import { AlertingSrv } from '@src/services';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useRequest } from '@src/hooks';
const { Meta } = Card;
const { Title, Text } = Typography;

const NotificationTypeSelect: React.FC<{ initType?: string; onChange: (type: string) => void }> = (props) => {
  const { initType, onChange } = props;
  const { boot } = useContext(PlatformContext);
  return (
    <Select
      defaultValue={initType}
      style={{ width: 160 }}
      onChange={(value: any) => {
        onChange(value);
      }}>
      {(boot.notifications || []).map((n: any, _idx: number) => {
        return (
          <Select.Option key={n.type} value={n.type} showTick={false}>
            <Text>{n.type}</Text>
          </Select.Option>
        );
      })}
    </Select>
  );
};

const Receiver: React.FC<{ receiver: any }> = (props) => {
  const { receiver } = props;
  const { boot } = useContext(PlatformContext);
  const notificationTypes = boot.notifications;
  const [notificationType, setNotificationType] = useState<any>(() => {
    return find(notificationTypes, { type: receiver.type });
  });
  const config = useRef<ReceiverConfig>();

  useMemo(() => {
    config.current = receiver;
  }, [receiver]);

  return (
    <Collapse.Panel
      showArrow={false}
      itemKey="test"
      header={
        <div className="item">
          <div className="desc">
            <Button icon={<IconChevronDown />} size="small" theme="borderless" type="tertiary" />
            <NotificationTypeSelect
              initType={receiver.type}
              onChange={(type: string) => {
                receiver.type = type;
                setNotificationType(find(notificationTypes, { type: type }));
              }}
            />
          </div>
          <div className="actions">
            <Button
              size="small"
              theme="borderless"
              type="tertiary"
              icon={<IconPlay />}
              onClick={() => {
                if (config.current) {
                  AlertingSrv.testNotify(config.current);
                }
              }}
            />
            <Button size="small" theme="borderless" type="tertiary" icon={<IconCopy />} />
            <Button size="small" theme="borderless" type="tertiary" icon={<IconDeleteStroked />} />
          </div>
        </div>
      }>
      <DynamicForm
        fields={get(notificationType, 'config', [])}
        initValues={receiver?.config}
        onValueChange={(values: any) => {
          if (config.current) {
            values.type = receiver.type;
            receiver.config = values;
            config.current = values;
          }
        }}
      />
    </Collapse.Panel>
  );
};

export const NotificationSetting: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [notification, setNotification] = useState({ name: '', receivers: [] as any[] });
  const uid = `${searchParams.get('uid')}`;
  const formApi = useRef<any>();
  const { loading, result } = useRequest(
    ['load-alert-notification', uid],
    async () => {
      return AlertingSrv.getNotification(uid);
    },
    { enable: !isEmpty(uid) }
  );
  useEffect(() => {
    setNotification(result);
  }, [result]);

  useEffect(() => {
    if (notification) {
      formApi.current.setValues(notification);
    }
  }, [notification]);

  return (
    <Card
      className="setting-page"
      bordered={false}
      bodyStyle={{ padding: '0px 24px 24px 24px' }}
      title={
        <Meta
          className="setting-meta"
          title={<Title heading={3}>Notification</Title>}
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Current organization:</Text>
            </div>
          }
          avatar={<Icon icon="alert" />}
        />
      }>
      <Form
        getFormApi={(api: any) => {
          formApi.current = api;
        }}
        onValueChange={(values: any) => {
          notification.name = values.name;
        }}>
        <Form.Input label="Name" field="name" rules={[{ required: true, message: 'Name is required' }]} />
      </Form>
      <Collapse
        activeKey={['test']}
        className="linsight-collapse"
        expandIconPosition="left"
        clickHeaderToExpand={false}>
        {(notification?.receivers || []).map((r: any, idx: number) => (
          <Receiver key={idx} receiver={r} />
        ))}
      </Collapse>
      <div style={{ display: 'flex', gap: 4 }}>
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            notification.receivers.push({ type: 'Mail' });
            setNotification(cloneDeep(notification));
          }}>
          New
        </Button>
        <Button
          icon={<IconSaveStroked />}
          onClick={() => {
            if (notification.uid) {
              AlertingSrv.updateNotification(notification);
            } else {
              AlertingSrv.createNotification(notification);
            }
          }}>
          Save
        </Button>
        <Button
          type="tertiary"
          onClick={() => {
            navigate({
              pathname: '/alerting/notifications',
            });
          }}>
          Cancel
        </Button>
      </div>
    </Card>
  );
};
