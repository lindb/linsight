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
import { Form } from '@douyinfe/semi-ui';
import { VariableEditorProps } from '@src/types';
import MetricNameSelect from './components/MetricNameSelect';
import TagKeySelect from './components/TagKeySelect';

const VariableEditor: React.FC<VariableEditorProps> = (props) => {
  const { variable, datasource } = props;
  console.log('variable editor', variable);
  return (
    <>
      <Form.Select
        field="query.request.type"
        label="Value type"
        style={{ width: 300 }}
        rules={[{ required: true, message: 'Value type is required.' }]}
        optionList={[
          { value: 'namespace', label: 'Namespace' },
          { value: 'metric', label: 'Metric' },
          { value: 'tagValue', label: 'Tag Value' },
        ]}
      />
      {variable.query?.request.valueType === 'tagValue' && (
        <>
          <MetricNameSelect
            labelPosition="top"
            label="Metric"
            field="query.request.metric"
            datasource={datasource}
            style={{ width: 300 }}
          />
          <TagKeySelect
            labelPosition="top"
            label="Tag key"
            field="query.request.tagKey"
            metricField="query.request.metric"
            datasource={datasource}
            style={{ width: 300 }}
          />
        </>
      )}
    </>
  );
};

export default VariableEditor;
