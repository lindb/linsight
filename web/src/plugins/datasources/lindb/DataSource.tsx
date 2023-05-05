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
import { DatasourceAPI, DatasourceSetting } from '@src/types';
import * as _ from 'lodash-es';

export class LinDBDatasource extends DatasourceAPI {
  constructor(setting: DatasourceSetting) {
    super(setting);
  }

  async fetchMetricNames(prefix?: string): Promise<string[]> {
    const rs = await DataQuerySrv.query({
      queries: [{ datasource: { uid: this.setting.uid }, request: { db: '_internal', sql: 'show metrics' } }],
    });
    if (rs) {
      return rs[0].values;
    }
    return [];
  }

  async getFields(metric: string): Promise<string[]> {
    if (_.isEmpty(metric)) {
      return [];
    }
    const rs = await DataQuerySrv.query({
      queries: [{ datasource: { uid: this.setting.uid }, request: { sql: `show fields from '${metric}'` } }],
    });
    if (rs) {
      return rs[0].values;
    }
    return [];
  }

  getTagKeys() {
    console.log('get tag keys');
  }

  query(req: any) {
    return DataQuerySrv.query({
      queries: [
        {
          datasource: { uid: this.setting.uid },
          request: { sql: `select go_threads from 'lindb.runtime' group by node,time()` },
        },
      ],
    });
  }

  test() {
    console.log('test....');
  }
}
