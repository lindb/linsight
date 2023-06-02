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
import { VariableContext } from '@src/contexts';
import { DataQuerySrv } from '@src/services';
import { DatasourceStore } from '@src/stores';
import { DataQuery, Query, TimeRange } from '@src/types';
import { isEmpty, cloneDeep } from 'lodash-es';
import { useContext } from 'react';
import { useRequest } from './use.request';

const getDataQuery = (queries: Query[], variables: object): DataQuery => {
  const dataQuery: DataQuery = { queries: [] };
  (queries || []).forEach((q: Query) => {
    if (q.hide || !q.datasource) {
      return;
    }
    const ds = DatasourceStore.getDatasource(q.datasource.uid);
    if (!ds) {
      return;
    }
    // NOTE: need clone new q, because rewrite query will modify it
    const queryAfterRewrite = ds.api.rewriteQuery(cloneDeep(q), variables);
    if (isEmpty(queryAfterRewrite)) {
      return;
    }
    // add query request into batch
    dataQuery.queries.push(queryAfterRewrite);
  });
  return dataQuery;
};

export const useMetric = (queries: Query[]) => {
  const { variables, from, to } = useContext(VariableContext);
  const dataQuery: DataQuery = getDataQuery(queries, variables);
  const { result, loading, refetch, error } = useRequest(
    ['query_metric_data', dataQuery, from, to], // watch dataQuery/from/to if changed
    async () => {
      const range: TimeRange = {};
      if (!isEmpty(from)) {
        range.from = `${from}`;
      }
      if (!isEmpty(to)) {
        range.to = `${to}`;
      }
      dataQuery.range = range;
      return DataQuerySrv.dataQuery(dataQuery);
    },
    { enabled: !isEmpty(dataQuery.queries) }
  );
  return {
    loading,
    result,
    error,
    refetch,
  };
};
