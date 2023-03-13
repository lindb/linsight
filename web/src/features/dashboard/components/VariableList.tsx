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
import { DashboardStore } from '@src/stores';
import * as _ from 'lodash-es';
import { Button, Table } from '@douyinfe/semi-ui';
import { IconPlusStroked, IconDeleteStroked, IconCopyStroked } from '@douyinfe/semi-icons';
import { StatusTip } from '@src/components';
import { observer } from 'mobx-react-lite';
import VariableEditor from './VariableEditor';

const VariableList: React.FC = () => {
  const { dashboard } = DashboardStore;
  const variables: any[] = _.get(dashboard, 'config.variables', []);
  return (
    <>
      <VariableEditor />
      <Button
        style={{ marginBottom: 12 }}
        icon={<IconPlusStroked />}
        onClick={() => DashboardStore.addVariable({ variable: `variable${variables.length}`, name: 'Variable' })}>
        New
      </Button>
      <Table
        size="small"
        pagination={false}
        bordered
        empty={<StatusTip isEmpty={_.isEmpty(variables)} />}
        dataSource={variables}
        columns={[
          { title: 'Variable', dataIndex: 'variable' },
          { title: 'Name', dataIndex: 'name' },
          { title: 'Definition' },
          {
            title: 'Operations',
            render: () => {
              return (
                <div>
                  <IconCopyStroked />
                  <IconDeleteStroked />
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
