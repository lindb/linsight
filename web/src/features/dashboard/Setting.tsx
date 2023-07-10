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
import React, { useContext, useEffect, useRef } from 'react';
import { Tabs, TabPane, Card, Form, Typography, Select } from '@douyinfe/semi-ui';
import { DashboardStore } from '@src/stores';
import Variables from './components/Variables';
import { observer } from 'mobx-react-lite';
import { useSearchParams } from 'react-router-dom';
import { PlatformContext } from '@src/contexts';
import { Integration } from '@src/types';
import { Icon } from '@src/components';
import { find, isEmpty } from 'lodash-es';

const { Text } = Typography;

const GeneralForm: React.FC = observer(() => {
  const { dashboard } = DashboardStore;
  const { boot } = useContext(PlatformContext);
  const formApi = useRef<any>();

  useEffect(() => {
    if (formApi.current) {
      formApi.current.setValues(dashboard);
    }
  }, [dashboard]);

  return (
    <Form
      labelPosition="top"
      style={{ maxWidth: 650 }}
      getFormApi={(api: any) => (formApi.current = api)}
      onValueChange={(values: any) => DashboardStore.updateDashboardProps(values)}>
      <Form.Input label="Title" field="title" />
      <Form.TextArea label="Description" field="desc" />
      <Form.Select
        style={{ width: 300 }}
        label="Integration"
        field="integration"
        filter={(input: string, option: any) => {
          if (isEmpty(input)) {
            return true;
          }
          return option.title.toUpperCase().includes(input.toUpperCase());
        }}
        showClear
        renderSelectedItem={(n: Record<string, any>) => {
          const integration = find(boot.integrations, { uid: n.value });
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Icon icon={`${integration?.icon}`} style={{ fontSize: 20 }} />
              <Text>{integration?.title}</Text>
            </div>
          );
        }}>
        {(boot.integrations || []).map((integration: Integration) => {
          return (
            <Select.Option
              style={{ display: 'flex', alignItems: 'center', gap: 6 }}
              key={integration.uid}
              value={integration.uid}
              title={integration.title}
              showTick={false}>
              <Icon icon={integration.icon} style={{ fontSize: 20 }} />
              <Text>{integration.title}</Text>
            </Select.Option>
          );
        })}
      </Form.Select>
    </Form>
  );
});

const Setting: React.FC<{}> = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  return (
    <Card className="linsight-feature dashboard-setting">
      <Tabs
        tabPaneMotion={false}
        tabPosition="left"
        defaultActiveKey={searchParams.get('tab') || 'general'}
        onChange={(activeKey: string) => {
          searchParams.set('tab', activeKey);
          setSearchParams(searchParams);
        }}>
        <TabPane tab="General" itemKey="general">
          <GeneralForm />
        </TabPane>
        <TabPane tab="Variables" itemKey="variables">
          <Variables />
        </TabPane>
        <TabPane tab="Permissions" itemKey="permissions">
          permissions
        </TabPane>
      </Tabs>
    </Card>
  );
};

export default Setting;
