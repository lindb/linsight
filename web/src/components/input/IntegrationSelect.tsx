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
import { Form, Select, Typography } from '@douyinfe/semi-ui';
import { PlatformContext } from '@src/contexts';
import { Integration } from '@src/types';
import { find, isEmpty } from 'lodash-es';
import React, { useContext } from 'react';
import Icon from '../common/Icon';

const { Text } = Typography;

const IntegrationSelect: React.FC<{}> = () => {
  const { boot } = useContext(PlatformContext);
  return (
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
  );
};

export default IntegrationSelect;
