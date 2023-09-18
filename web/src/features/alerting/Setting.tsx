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
import { Button, Card, Table, Typography } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import React, { useContext, useMemo, useRef } from 'react';
import { DynamicForm, Icon } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { useRequest } from '@src/hooks';
import { AlertingSrv } from '@src/services';
import { NotificationPlugin, NotificationSetting } from '@src/types';
import { find } from 'lodash-es';
const { Meta } = Card;
const { Title, Text } = Typography;

const SettingForm: React.FC<{ plugin: NotificationPlugin; setting?: NotificationSetting }> = (props) => {
  const { plugin, setting } = props;
  const config = useRef<NotificationSetting>();

  useMemo(() => {
    config.current = setting;
  }, [setting]);

  return (
    <div>
      <DynamicForm
        initValues={setting}
        layout="horizontal"
        fields={plugin.setting}
        onValueChange={(values: any) => {
          config.current = values;
        }}
      />
      <Button
        icon={<IconSaveStroked />}
        onClick={() => {
          if (config.current) {
            config.current.type = plugin.type;
            AlertingSrv.saveNotificationSetting(config.current);
          }
        }}>
        Save
      </Button>
    </div>
  );
};

export const Setting: React.FC = () => {
  const { boot } = useContext(PlatformContext);
  const notificationTypes = boot.notifications;
  const { result: settings } = useRequest(['load_notification_settings'], () => {
    return AlertingSrv.getAllNotificationSettings();
  });
  return (
    <Card
      className="setting-page"
      bordered={false}
      title={
        <Meta
          className="setting-meta"
          title={<Title heading={3}>Setting</Title>}
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Current organization:</Text>
            </div>
          }
          avatar={<Icon icon="setting" />}
        />
      }>
      <Table
        bordered
        size="small"
        dataSource={notificationTypes || []}
        rowKey="type"
        columns={[
          {
            title: 'Type',
            dataIndex: 'type',
          },
          {
            title: 'Enable',
            dataIndex: 'enable',
          },
        ]}
        pagination={false}
        expandRowByClick
        expandedRowRender={(row: any) => {
          return (
            <SettingForm
              plugin={row as NotificationPlugin}
              setting={find(settings, { type: row.type }) as NotificationSetting}
            />
          );
        }}
      />
    </Card>
  );
};
