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
import { Variable, VariableHideType } from '@src/types';
import { useRequest } from './use.request';
import { isEmpty } from 'lodash-es';
import { DataQuerySrv } from '@src/services';
import { VariableContext } from '@src/contexts';
import { useContext } from 'react';

export const useVariable = (variable: Variable | undefined, prefix?: string) => {
  const query = variable?.query;
  const { variables } = useContext(VariableContext);
  const { result, loading, refetch, error } = useRequest(
    ['search_metric_metadata', query],
    async () => {
      if (isEmpty(query)) {
        return [];
      }
      // get datasource by uid
      const ds = DatasourceStore.getDatasource(query.datasource.uid);
      if (!ds) {
        return [];
      }
      const q = ds.api.rewriteMetaQuery(query, variables, prefix);
      if (!q) {
        return [];
      }
      const rs = await DataQuerySrv.metadataQuery(q);
      if (rs) {
        return rs.values;
      }
      return [];
    },
    {
      enabled: variable?.hide !== VariableHideType.Hide && !isEmpty(query),
    }
  );
  return {
    loading,
    result,
    error,
    refetch,
  };
};
