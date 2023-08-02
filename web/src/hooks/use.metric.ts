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
import { MixedDatasource } from '@src/constants';
import { VariableContext } from '@src/contexts';
import { DataQuerySrv } from '@src/services';
import { DatasourceStore } from '@src/stores';
import { DataQuery, DataSetType, Query, TimeRange } from '@src/types';
import { TimeKit } from '@src/utils';
import { isEmpty, cloneDeep, get } from 'lodash-es';
import { useContext } from 'react';
import { useRequest } from './use.request';

const getDataQuery = (
  queries: Query[],
  variables: object,
  dataset: DataSetType,
  defaultDatasourceUID?: string
): DataQuery => {
  const dataQuery: DataQuery = { queries: [] };
  (queries || []).forEach((q: Query) => {
    if (q.hide) {
      return;
    }
    let datasourceUID = get(q.datasource, 'uid');
    if (isEmpty(datasourceUID)) {
      datasourceUID = `${defaultDatasourceUID}`;
    }
    if (datasourceUID === MixedDatasource) {
      // ignore mixed datasource
      return;
    }
    const ds = DatasourceStore.getDatasource(datasourceUID);
    if (!ds) {
      return;
    }
    // NOTE: need clone new q, because rewrite query will modify it
    const queryAfterRewrite = ds.api.rewriteQuery(cloneDeep(q), variables, dataset);
    if (isEmpty(queryAfterRewrite)) {
      return;
    }
    // need set datasource, maybe target no datasource setting, using default datasource
    queryAfterRewrite.datasource = { uid: datasourceUID };
    // add query request into batch
    dataQuery.queries.push(queryAfterRewrite);
  });
  return dataQuery;
};

export const useMetric = (queries: Query[], dataset: DataSetType, defaultDatasourceUID?: string) => {
  const { variables, from, to, refreshTime } = useContext(VariableContext);
  const dataQuery: DataQuery = getDataQuery(queries, variables, dataset, defaultDatasourceUID);
  const { result, loading, refetch, error } = useRequest(
    ['query_metric_data', dataQuery, from, to, refreshTime], // watch dataQuery/from/to if changed
    async () => {
      const range: TimeRange = {};
      if (!isEmpty(from)) {
        range.from = TimeKit.parseTime(from);
      }
      if (!isEmpty(to)) {
        range.to = TimeKit.parseTime(to);
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
