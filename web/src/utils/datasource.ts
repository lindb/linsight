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
import { DatasourceStore } from '@src/stores';
import { DatasourceCategory, DatasourceInstance } from '@src/types';
import { set } from 'lodash-es';
import { createSearchParams } from 'react-router-dom';

const isTrace = (datasource: DatasourceInstance | null | undefined): boolean => {
  if (datasource && datasource.plugin) {
    return datasource.plugin.category === DatasourceCategory.Trace;
  }
  return false;
};

const isMetric = (datasource: DatasourceInstance | null | undefined): boolean => {
  if (datasource && datasource.plugin) {
    return datasource.plugin.category === DatasourceCategory.Metric;
  }
  return false;
};

const getDatasourceDefaultParams = (datasourceUID: string): string => {
  const datasource = DatasourceStore.getDatasource(datasourceUID);
  if (!datasource) {
    return '';
  }
  const params = datasource?.plugin.getDefaultParams();
  set(params, 'datasource', { uid: datasource?.setting.uid, type: datasource?.plugin.Type });
  return createSearchParams({ left: JSON.stringify(params) }).toString();
};

export default {
  isTrace,
  isMetric,
  getDatasourceDefaultParams,
};
