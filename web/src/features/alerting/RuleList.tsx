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
import { Card, Table, Button, Typography } from '@douyinfe/semi-ui';
import { IconPlusStroked } from '@douyinfe/semi-icons';
import { SearchFilterInput } from '@src/components';
import { useRequest } from '@src/hooks';
import { AlertingSrv } from '@src/services';
import React from 'react';
import { createSearchParams, useNavigate } from 'react-router-dom';
const { Text } = Typography;

export const RuleList: React.FC = () => {
  const navigate = useNavigate();
  const { result: rules } = useRequest(['search_alert_rules'], () => {
    return AlertingSrv.searchAlertRules({});
  });
  return (
    <Card className="setting-page" bordered={false}>
      <div style={{ margin: 16, display: 'flex', gap: 8 }}>
        <SearchFilterInput placeholder="Filter rule" />
        <Button size="large" icon={<IconPlusStroked />} onClick={() => navigate('/alerting/rules/new')}>
          New
        </Button>
      </div>
      <Table
        bordered
        size="small"
        dataSource={rules?.rules || []}
        rowKey="uid"
        columns={[
          {
            title: 'Title',
            dataIndex: 'title',
            render: (_text: any, r: any, _index: any) => {
              return (
                <div
                  className="dashboard-title"
                  onClick={() => {
                    navigate({
                      pathname: '/alerting/rules/edit',
                      search: `${createSearchParams({
                        uid: `${r.uid}`,
                      })}`,
                    });
                  }}>
                  <Text link>{r.title}</Text>
                </div>
              );
            },
          },
          {
            title: 'Enable',
            dataIndex: 'enable',
          },
        ]}
      />
    </Card>
  );
};
