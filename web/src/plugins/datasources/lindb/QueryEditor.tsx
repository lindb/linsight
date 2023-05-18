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
import { Divider, Row, Col, Form, Typography } from '@douyinfe/semi-ui';
import { LinSelect } from '@src/components';
import { DatasourceInstance, QueryEditorProps } from '@src/types';
import React, { useContext } from 'react';
import { get, clone } from 'lodash-es';
import { LinDBDatasource } from './Datasource';
import { QueryEditContext } from '@src/contexts';
import { useQueryEditor } from '@src/hooks';
import MetricNameSelect from './components/MetricNameSelect';
import TagKeySelect from './components/TagKeySelect';
import FieldSelect from './components/FieldSelect';
import WhereConditonSelect from './components/WhereEditor';

const { Text } = Typography;

const FieldEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource
  const { metric } = useQueryEditor(['metric']);

  return (
    <LinSelect
      style={{ width: 240 }}
      field="fields"
      placeholder="Please select fields"
      labelPosition="inset"
      label="Fields"
      multiple
      showClear
      filter
      remote
      reloadKeys={[metric]}
      loader={async () => {
        const values = await api.getFields(metric);
        const optionList: any[] = [];
        (values || []).map((item: any) => {
          optionList.push({ value: item.name, label: `${item.name}(${item.type})` });
        });
        return optionList;
      }}
    />
  );
};

const QueryEditor: React.FC<QueryEditorProps> = (props) => {
  const { datasource, onChange } = props;
  const { values, setValues } = useContext(QueryEditContext);
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource
  const metric = get(values, 'metric');
  console.log('change metric....', metric);

  return (
    <Form
      layout="horizontal"
      initValues={values}
      onSubmit={(values: any) => {
        console.log('query request', values);
        if (onChange) {
          // TODO: check params if changed.
          // FIXME: clone just triger change state
          onChange(clone(values));
          setValues(values);
        }
      }}>
      <MetricNameSelect datasource={datasource} style={{ width: 240 }} />
      <FieldSelect datasource={datasource} style={{ width: 240 }} />
      <WhereConditonSelect datasource={datasource} />
      <TagKeySelect
        field="groupBy"
        placeholder="Please select group by tag key"
        multiple
        label="Group By"
        datasource={datasource}
      />
    </Form>
  );
};

export default QueryEditor;
