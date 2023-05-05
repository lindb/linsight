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
import { Button, Form, Radio } from '@douyinfe/semi-ui';
import { IconTick } from '@douyinfe/semi-icons';
import { DatasourceRepositoryInst } from '@src/types';

const VariableEditor: React.FC = () => {
  const QueryEditor = () => {
    const plugin = DatasourceRepositoryInst.get('lindb');
    if (!plugin) {
      return null;
    }
    const VariableQueryEditor = plugin.components.VariableEditor;
    if (!VariableQueryEditor) {
      return null;
    }
    return <VariableQueryEditor />;
  };
  return (
    <Form className="linsight-form" labelPosition="top" extraTextPosition="middle" initValues={{ type: 'query' }}>
      <Form.Input
        field="name"
        label="Name"
        extraText="The name of variable"
        rules={[{ required: true, message: 'Name is required.' }]}
      />
      <Form.Input field="label" label="Label" extraText="Display name(optional)" />
      <Form.RadioGroup field="hide" label="Show" type="button">
        <Radio value="1">Label and Name</Radio>
        <Radio value="2">Name</Radio>
        <Radio value="3">Nothing</Radio>
      </Form.RadioGroup>
      <Form.Select
        field="type"
        label="Type"
        rules={[{ required: true, message: 'Name is required.' }]}
        optionList={[
          { value: 'query', label: 'Query' },
          { value: 'constant', label: 'Constant' },
        ]}
      />
      <QueryEditor />
      <Form.Slot>
        <Button type="primary" icon={<IconTick />} htmlType="submit">
          Apply
        </Button>
      </Form.Slot>
    </Form>
  );
};

export default VariableEditor;
