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
import { useFormApi } from '@douyinfe/semi-ui';
import { LinSelect } from '@src/components';
import React, { CSSProperties } from 'react';
import { LinDBDatasource } from '../Datasource';

/*
 * Namespace select for query/variable editor.
 */
const NamespaceSelect: React.FC<{
  datasource: LinDBDatasource;
  label?: string;
  field?: string;
  style?: CSSProperties;
  labelPosition?: 'top' | 'left' | 'inset';
}> = (props) => {
  const { datasource, label, style, field = 'namespace', labelPosition } = props;
  const formApi = useFormApi();
  return (
    <LinSelect
      style={style}
      label={label}
      field={field}
      placeholder="Please select namespace"
      labelPosition={labelPosition}
      resetValue={{ metric: '', fields: null, where: null, groupBy: null }}
      loader={async (prefix?: string) => {
        const values = await datasource.fetchNamespaces(prefix);
        const optionList: any[] = [];
        (values || []).map((item: any) => {
          optionList.push({ value: item, label: item, showTick: false });
        });
        return optionList;
      }}
      onFinished={() => formApi.submitForm()}
    />
  );
};

export default NamespaceSelect;
