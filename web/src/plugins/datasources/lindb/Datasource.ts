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
import { DataQuerySrv } from '@src/services';
import { DataSetType, DatasourceAPI, DatasourceSetting, Query, TimeRange } from '@src/types';
import { TemplateKit } from '@src/utils';
import { isArray, isEmpty, isString } from 'lodash-es';
import { ConditionExpr, Operator } from './types';

export class LinDBDatasource extends DatasourceAPI {
  constructor(setting: DatasourceSetting) {
    super(setting);
  }

  rewriteQuery(query: Query, variables: {}, dataset: DataSetType): Query | null {
    if (!query.request || isEmpty(query.request.metric) || isEmpty(query.request.fields)) {
      return null;
    }
    if (!isEmpty(query.request.where)) {
      const where: any[] = [];
      query.request.where.forEach((w: any) => {
        if (isString(w.value)) {
          w.value = TemplateKit.template(w.value, variables);
        } else if (isArray(w.value)) {
          const newValues: string[] = [];
          w.value.forEach((v: string) => {
            const newVal = TemplateKit.template(v, variables);
            if (isEmpty(newVal)) {
              return;
            } else if (isArray(newVal)) {
              newValues.push(...newVal);
            } else {
              newValues.push(newVal);
            }
          });
          w.value = newValues;
        }
        if (isEmpty(w.value) && w.optional) {
          // ignore empty condition, if it is optional
          return;
        }
        where.push(w);
      });
      query.request.where = where;
    }
    if (dataset !== DataSetType.TimeSeries) {
      query.request.stats = true;
    } else {
      query.request.stats = false;
    }
    return query;
  }

  async fetchMetricNames(namespace: string, prefix?: string): Promise<string[]> {
    const req: any = {
      type: 'metric',
      namespace: namespace,
    };
    if (!isEmpty(prefix)) {
      req.where = [{ key: 'metric', operator: Operator.Eq, value: prefix }];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: req,
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async fetchNamespaces(prefix?: string): Promise<string[]> {
    const req: any = {
      type: 'namespace',
    };
    if (!isEmpty(prefix)) {
      req.where = [{ key: 'namespace', operator: Operator.Eq, value: prefix }];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: req,
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getFields(namespace: string, metric: string): Promise<string[]> {
    if (isEmpty(metric)) {
      return [];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'field', metric: metric, namespace: namespace },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getTagKeys(namespace: string, metric: string): Promise<string[]> {
    if (isEmpty(metric)) {
      return [];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'tagKey', metric: metric, namespace: namespace },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getTagValues(
    namespace: string,
    metric: string,
    tagKey: string,
    conditions: ConditionExpr[] = [],
    prefix?: string
  ): Promise<string[]> {
    if (isEmpty(metric) || isEmpty(tagKey)) {
      return [];
    }
    const req: any = {
      type: 'tagValue',
      namespace: namespace,
      metric: metric,
      tagKey: tagKey,
      where: conditions,
    };
    if (!isEmpty(prefix)) {
      conditions.push({ key: tagKey, operator: Operator.Like, value: `${prefix}*` });
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: req,
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async metaQuery(req: any, prefix?: string): Promise<string[]> {
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: req,
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  query(req: any, range?: TimeRange) {
    return DataQuerySrv.dataQuery({
      range: range,
      queries: [
        {
          datasource: { uid: this.setting.uid },
          request: req,
        },
      ],
    });
  }

  test() {
    console.log('test....');
  }
}
