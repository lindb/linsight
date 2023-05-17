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
import { LinSelect } from '@src/components';
import { DatasourceInstance } from '@src/types';
import React, { CSSProperties } from 'react';
import { LinDBDatasource } from '../Datasource';

const MetricNameSelect: React.FC<{
  datasource: DatasourceInstance;
  field?: string;
  style?: CSSProperties;
  labelPosition?: 'top' | 'left' | 'inset';
}> = (props) => {
  const { datasource, style, field = 'metric', labelPosition } = props;
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource
  return (
    <LinSelect
      style={style}
      field={field}
      placeholder="Please select metric"
      labelPosition={labelPosition}
      label="Metric"
      showClear
      filter
      remote
      loader={async (prefix?: string) => {
        console.log('load metric name......');
        const values = await api.fetchMetricNames(prefix);
        const optionList: any[] = [];
        console.log('kslfasjdflkdsj', values, optionList);
        (values || []).map((item: any) => {
          optionList.push({ value: item, label: item });
        });
        return optionList;
      }}
    />
  );
};

export default MetricNameSelect;
