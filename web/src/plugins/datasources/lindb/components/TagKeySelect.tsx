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
import React, { CSSProperties } from 'react';
import { LinSelect } from '@src/components';
import { LinDBDatasource } from '../Datasource';
import { useFieldState } from '@douyinfe/semi-ui';

const TagKeySelect: React.FC<{
  datasource: LinDBDatasource;
  field?: string;
  label?: string;
  style?: CSSProperties;
  multiple?: boolean;
  metricField?: string;
  placeholder?: string;
  labelPosition?: 'top' | 'left' | 'inset';
  onFinished?: () => void;
}> = (props) => {
  const {
    datasource,
    multiple,
    label,
    style,
    placeholder = 'Please select tag key',
    field = 'tagKey',
    metricField = 'metric',
    labelPosition,
    onFinished,
  } = props;
  const { value: metricName } = useFieldState(metricField);
  return (
    <LinSelect
      style={style}
      field={field}
      label={label}
      multiple={multiple}
      placeholder={placeholder}
      labelPosition={labelPosition}
      reloadKeys={[metricField]}
      loader={async (_prefix?: string) => {
        const values = await datasource.getTagKeys(metricName);
        const optionList: any[] = [];
        (values || []).map((item: any) => {
          optionList.push({ value: item, label: item, showTick: false });
        });
        return optionList;
      }}
      onFinished={onFinished}
    />
  );
};

export default TagKeySelect;
