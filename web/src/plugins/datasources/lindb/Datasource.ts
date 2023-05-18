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
import { DatasourceAPI, DatasourceSetting, TimeRange } from '@src/types';
import { isEmpty } from 'lodash-es';
import { toJS } from 'mobx';

export class LinDBDatasource extends DatasourceAPI {
  constructor(setting: DatasourceSetting) {
    super(setting);
  }

  async fetchMetricNames(prefix?: string): Promise<string[]> {
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'metric', prefix: prefix },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getFields(metric: string): Promise<string[]> {
    if (isEmpty(metric)) {
      return [];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'field', metric: metric },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getTagKeys(metric: string): Promise<string[]> {
    if (isEmpty(metric)) {
      return [];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'tagKey', metric: metric },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async getTagValues(metric: string, tagKey: string, prefix?: string): Promise<string[]> {
    if (isEmpty(metric) || isEmpty(tagKey)) {
      return [];
    }
    const rs = await DataQuerySrv.metadataQuery({
      datasource: { uid: this.setting.uid },
      request: { type: 'tagValue', metric: metric, tagKey: tagKey, prefix: prefix },
    });
    if (rs) {
      return rs.values;
    }
    return [];
  }

  async metaQuery(req: any, prefix?: string): Promise<string[]> {
    console.log('req metadata', toJS(req));
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
    console.log(req, 'xxxxxx');
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
