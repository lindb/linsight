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
import { LinDBDatasource } from './Datasource';
import { compact, get, isEmpty } from 'lodash-es';
import WhereConditonSelect from './components/WhereEditor';
import NamespaceSelect from './components/NamespaceSelect';

const VariableEditor: React.FC<VariableEditorProps> = (props) => {
  const { variable, datasource } = props;
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource
  const namespace = get(datasource, 'setting.config.namespace', '');
  return (
    <>
      <Form.Select
        field="query.request.type"
        label="Value type"
        style={{ width: 300 }}
        rules={[{ required: true, message: 'Value type is required.' }]}
        optionList={compact([
          isEmpty(namespace) && { value: 'namespace', label: 'Namespace', showTick: false },
          { value: 'metric', label: 'Metric', showTick: false },
          { value: 'tagValue', label: 'Tag Value', showTick: false },
        ])}
      />
      {(get(variable, 'query.request.type') === 'metric' || get(variable, 'query.request.type') === 'tagValue') &&
        isEmpty(namespace) && (
          <NamespaceSelect
            labelPosition="top"
            field="query.request.namespace"
            label={get(datasource, 'setting.config.alias', 'Namespace')}
            datasource={api}
            style={{ minWidth: 300 }}
          />
        )}
      {get(variable, 'query.request.type') === 'tagValue' && (
        <>
          <MetricNameSelect
            labelPosition="top"
            label="Metric"
            field="query.request.metric"
            datasource={api}
            ns={namespace}
            style={{ width: 300 }}
          />
          <TagKeySelect
            labelPosition="top"
            label="Tag key"
            field="query.request.tagKey"
            metricField="query.request.metric"
            ns={namespace}
            datasource={api}
            style={{ width: 300, marginBottom: 4 }}
          />
          <WhereConditonSelect
            field="query.request.where"
            ns={namespace}
            datasource={api}
            metricField="query.request.metric"
          />
        </>
      )}
    </>
  );
};

export default VariableEditor;
