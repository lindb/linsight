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
import { Card, Button, Table, Typography, Tag } from '@douyinfe/semi-ui';
import { IconPlusStroked } from '@douyinfe/semi-icons';
import { SearchFilterInput } from '@src/components';
import { useRequest } from '@src/hooks';
import { AlertingSrv } from '@src/services';
import React from 'react';
import { createSearchParams, useNavigate } from 'react-router-dom';

const { Text } = Typography;

export const NotificationList: React.FC = () => {
  const navigate = useNavigate();
  const { result: notifications } = useRequest(['search_notifications'], () => {
    return AlertingSrv.searchNotifications({});
  });
  return (
    <Card className="linsight-feature" bodyStyle={{ padding: 0 }}>
      <div style={{ margin: 16, display: 'flex', gap: 8 }}>
        <SearchFilterInput placeholder="Filter notification" />
        <Button size="large" icon={<IconPlusStroked />} onClick={() => navigate('/alerting/notifications/new')}>
          New
        </Button>
      </div>
      <Table
        bordered
        size="small"
        dataSource={notifications?.notifications || []}
        rowKey="uid"
        columns={[
          {
            title: 'Name',
            dataIndex: 'name',
            render: (_text: any, r: any, _index: any) => {
              return (
                <div
                  className="dashboard-title"
                  onClick={() => {
                    navigate({
                      pathname: '/alerting/notifications/edit',
                      search: `${createSearchParams({
                        uid: `${r.uid}`,
                      })}`,
                    });
                  }}>
                  <Text link>{r.name}</Text>
                </div>
              );
            },
          },
          {
            title: 'Receivers',
            dataIndex: 'receivers',
            render: (_text: any, r: any, _index: any) => {
              return (
                <div style={{ display: 'flex', gap: 4 }}>
                  {(r.receivers || []).map((receiver: any, idx: number) => {
                    return <Tag key={idx}>{receiver.type}</Tag>;
                  })}
                </div>
              );
            },
          },
        ]}
      />
    </Card>
  );
};
