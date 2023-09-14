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
import { Form } from '@douyinfe/semi-ui';
import React from 'react';

export const SettingEditor: React.FC = () => {
  return (
    <>
      <Form.Section text="Setting">
        <Form.Input
          field="url"
          label="URL"
          rules={[{ required: true, message: 'URL is required' }]}
          placeholder="http://localhost:9000"
        />
        <Form.Input
          field="config.database"
          label="Database"
          rules={[{ required: true, message: 'Database is required' }]}
        />
        <Form.InputGroup style={{ display: 'flex', gap: 4 }} label={{ text: 'Namespace', align: 'right' }}>
          <Form.Input field="config.namespace" noLabel style={{ width: 340 }} placeholder="Namespace" />
          <Form.Input field="config.alias" noLabel style={{ flex: 1 }} placeholder="Namespace alias" />
        </Form.InputGroup>
        <Form.Switch field="config.exemplar" label="Exemplar" />
      </Form.Section>
    </>
  );
};
