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
import React, { useState } from 'react';
import { DashboardStore } from '@src/stores';
import { get, isEmpty, cloneDeep } from 'lodash-es';
import { Button, Table, Typography } from '@douyinfe/semi-ui';
import { IconPlusStroked, IconDeleteStroked, IconCopyStroked, IconEyeOpened } from '@douyinfe/semi-icons';
import { StatusTip } from '@src/components';
import { observer } from 'mobx-react-lite';
import { useSearchParams } from 'react-router-dom';
import ViewVariables from './ViewVariables';
const { Text } = Typography;

const VariableList: React.FC = () => {
  const { dashboard } = DashboardStore;
  const [searchParams, setSearchParams] = useSearchParams();
  const variables: any[] = get(dashboard, 'config.variables', []);
  const [preview, setPreview] = useState(false);
  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'row', gap: 4 }}>
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            const len = variables.length;
            const variable = `variable${len}`;
            DashboardStore.addVariable({ name: variable, label: 'Variable' });
            searchParams.set('edit', `${len}`);
            setSearchParams(searchParams);
          }}>
          New
        </Button>
        <Button
          type="secondary"
          icon={<IconEyeOpened />}
          onClick={() => {
            setPreview(!preview);
          }}>
          Preview
        </Button>
      </div>
      {preview && (
        <div style={{ marginTop: 12 }}>
          <ViewVariables />
        </div>
      )}
      <Table
        style={{ marginTop: 12 }}
        size="small"
        pagination={false}
        bordered
        empty={<StatusTip isEmpty={isEmpty(variables)} />}
        dataSource={variables}
        columns={[
          {
            title: 'Name',
            dataIndex: 'name',
            render: (_text: any, r: any, index: number) => {
              return (
                <Text
                  link
                  onClick={() => {
                    searchParams.set('edit', `${index}`);
                    setSearchParams(searchParams);
                  }}>
                  {r.name}
                </Text>
              );
            },
          },
          { title: 'Label', dataIndex: 'label' },
          { title: 'Definition' },
          {
            title: 'Operations',
            render: (_text: any, r: any, index: number) => {
              return (
                <div style={{ display: 'flex', flexDirection: 'row', gap: 4 }}>
                  <Button
                    type="primary"
                    icon={<IconCopyStroked size="large" />}
                    onClick={() => {
                      const newVariable = cloneDeep(r);
                      newVariable.name = `copy_of_${newVariable.name}`;
                      DashboardStore.addVariable(newVariable);
                    }}
                  />
                  <Button
                    type="danger"
                    icon={<IconDeleteStroked size="large" />}
                    onClick={() => DashboardStore.deleteVariable(`${index}`)}
                  />
                </div>
              );
            },
          },
        ]}
      />
    </>
  );
};

export default observer(VariableList);
