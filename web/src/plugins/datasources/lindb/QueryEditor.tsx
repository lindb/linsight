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
import { QueryEditorProps } from '@src/types';
import React, { useContext, useRef } from 'react';
import { isEqual, cloneDeep } from 'lodash-es';
import { LinDBDatasource } from './Datasource';
import { QueryEditContext } from '@src/contexts';
import MetricNameSelect from './components/MetricNameSelect';
import TagKeySelect from './components/TagKeySelect';
import FieldSelect from './components/FieldSelect';
import WhereConditonSelect from './components/WhereEditor';
import './query-edit.scss';
import { ObjectKit } from '@src/utils';
import NamespaceSelect from './components/NamespaceSelect';

const QueryEditor: React.FC<QueryEditorProps> = (props) => {
  const { datasource } = props;
  const { values: initValues, setValues } = useContext(QueryEditContext);
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource
  const previous = useRef(cloneDeep(initValues));

  return (
    <Form
      className="lindb-query-editor"
      layout="horizontal"
      initValues={initValues}
      onSubmit={(values: any) => {
        const newValues = ObjectKit.cleanEmptyProperties(values);
        if (!isEqual(previous.current, newValues)) {
          previous.current = cloneDeep(newValues);
          // change query edit context's values
          setValues(newValues);
        }
      }}>
      {({ formApi }) => (
        <>
          <NamespaceSelect datasource={api} style={{ minWidth: 240 }} />
          <MetricNameSelect datasource={api} style={{ minWidth: 240 }} />
          <FieldSelect datasource={api} style={{ minWidth: 240 }} />
          <WhereConditonSelect datasource={api} style={{ minWidth: 270 }} />
          <TagKeySelect
            field="groupBy"
            placeholder="Please select group by tag key"
            multiple
            label="Group By"
            datasource={api}
            onFinished={() => formApi.submitForm()}
          />
        </>
      )}
    </Form>
  );
};

export default QueryEditor;
