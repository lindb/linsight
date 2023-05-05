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
import { Card, Form } from '@douyinfe/semi-ui';
import { DashboardStore } from '@src/stores';
import React from 'react';
import { get, isEmpty } from 'lodash-es';
import { observer } from 'mobx-react-lite';
import { VariableHideType } from '@src/types';
import './variables.scss';

const ViewVariables: React.FC<{ className?: string }> = (props) => {
  const { className } = props;
  const { dashboard } = DashboardStore;
  const variables: any[] = get(dashboard, 'config.variables', []);
  if (isEmpty(variables)) {
    return null;
  }
  return (
    <Card className={className} bodyStyle={{ padding: 8 }}>
      <Form labelPosition="inset" layout="horizontal" className="lin-variables">
        {variables.map((item) => {
          if (item.hide === VariableHideType.Hide) {
            return null;
          }
          return (
            <Form.Select
              key={item.name}
              noLabel={item.hide === VariableHideType.OnlyValue}
              label={item.label}
              field={item.name}
            />
          );
        })}
      </Form>
    </Card>
  );
};

export default observer(ViewVariables);
