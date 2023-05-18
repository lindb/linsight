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
import { DatasourceStore } from '@src/stores';
import { Query, SearchParamKeys, TimeRange } from '@src/types';
import { TemplateKit } from '@src/utils';
import { isEmpty, cloneDeep } from 'lodash-es';
import { toJS } from 'mobx';
import { useContext } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useRequest } from './use.request';

export const useMetric = (queries: Query[]) => {
  console.log('use metric.......', toJS(queries));
  const [searchParams] = useSearchParams();
  const from = searchParams.get(SearchParamKeys.From);
  const to = searchParams.get(SearchParamKeys.To);
  const { values } = useContext(VariableContext);
  console.log('use metric, variable values', values);
  const { result, loading, refetch, error } = useRequest(
    ['search_metric_data', queries, from, to, values],
    async () => {
      const requests: any[] = [];
      const range: TimeRange = {};
      if (!isEmpty(from)) {
        range.from = `${from}`;
      }
      if (!isEmpty(to)) {
        range.to = `${to}`;
      }
      (queries || []).forEach((q: Query) => {
        const query = cloneDeep(q);
        console.log(toJS(query), 'query.....');
        if (!isEmpty(query.request.where)) {
          query.request.where = query.request.where.map((w: any) => {
            w.value = TemplateKit.template(w.value, values);
            console.log('template', TemplateKit.template(w.value), w.value, values || { node: '1231231' });
            return w;
          });
        }
        const ds = DatasourceStore.getDatasource(query.datasource.uid);
        if (!ds) {
          return;
        }
        // add query request into batch
        requests.push(ds.api.query(query.request, range));
      });
      return Promise.allSettled(requests).then((res) => {
        return res.map((item) => (item.status === 'fulfilled' ? item.value : [])).flat();
      });
    },
    { enabled: !isEmpty(queries) }
  );
  return {
    loading,
    result,
    error,
    refetch,
  };
};
