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
import { DataSetType, DatasourceAPI, DatasourceSetting, Query } from '@src/types';
import { TemplateKit } from '@src/utils';
import { isArray, isEmpty, isString, compact } from 'lodash-es';

export class LinGoDatasource extends DatasourceAPI {
  constructor(setting: DatasourceSetting) {
    super(setting);
  }

  rewriteQuery(query: Query, _variables: {}, _dataset?: DataSetType): Query | null {
    if (!query.request) {
      return null;
    }
    return query;
  }

  rewriteMetaQuery(query: Query, variables: {}, prefix?: string): Query | null {
    return null;
  }

  findVariableNames(query: Query): string[] {
    if (isEmpty(query.request.where)) {
      return [];
    }
    const names: string[] = [];
    query.request.where.forEach((w: any) => {
      if (isString(w.value)) {
        const name = TemplateKit.findTemplateName(w.value);
        names.push(name);
      } else if (isArray(w.value)) {
        w.value.forEach((v: string) => {
          const name = TemplateKit.findTemplateName(v);
          names.push(name);
        });
      }
    });
    return compact(names);
  }

  rewriteWhereCondition(query: Query, variables: {}) {
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
  }

  test() {
    console.log('test....');
  }
}
