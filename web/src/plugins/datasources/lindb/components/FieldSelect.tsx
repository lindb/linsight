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
import { useFieldState, useFormApi } from '@douyinfe/semi-ui';
import { LinSelect } from '@src/components';
import React, { CSSProperties } from 'react';
import { LinDBDatasource } from '../Datasource';

/*
 * Field select for query/variable editor.
 */
const FieldSelect: React.FC<{
  datasource: LinDBDatasource;
  ns?: string;
  field?: string;
  label?: string;
  style?: CSSProperties;
  namespaceField?: string;
  metricField?: string;
  labelPosition?: 'top' | 'left' | 'inset';
}> = (props) => {
  const {
    datasource,
    ns,
    label,
    style,
    field = 'fields',
    namespaceField = 'namespace',
    metricField = 'metric',
    labelPosition,
  } = props;
  const { value: metricName } = useFieldState(metricField);
  const { value: namespace } = useFieldState(namespaceField);
  const formApi = useFormApi();
  return (
    <LinSelect
      style={style}
      field={field}
      label={label}
      multiple
      placeholder="Please select fields"
      labelPosition={labelPosition}
      reloadKeys={[metricField, namespaceField]}
      loader={async (_prefix?: string) => {
        const values = await datasource.getFields(ns || namespace, metricName);
        const optionList: any[] = [];
        (values || []).map((item: any) => {
          optionList.push({ value: item.name, label: `${item.name}(${item.type})`, showTick: false });
        });
        return optionList;
      }}
      onFinished={() => formApi.submitForm()}
    />
  );
};

export default FieldSelect;
