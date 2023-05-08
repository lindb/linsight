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
import { ApiPath } from '@src/constants';
import { DatasourceSetting } from '@src/types';
import { ApiKit } from '@src/utils';

const createDatasource = (ds: DatasourceSetting): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Datasource, ds);
};

const updateDatasource = (ds: DatasourceSetting): Promise<DatasourceSetting> => {
  return ApiKit.PUT<DatasourceSetting>(ApiPath.Datasource, ds);
};

const getDatasource = (uid: string): Promise<DatasourceSetting> => {
  return ApiKit.GET<DatasourceSetting>(`${ApiPath.Datasources}/${uid}`);
};

const deleteDatasource = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Datasources}/${uid}`);
};

const fetchDatasources = (): Promise<DatasourceSetting[]> => {
  return ApiKit.GET<DatasourceSetting[]>(ApiPath.Datasources);
};

export default {
  createDatasource,
  updateDatasource,
  getDatasource,
  deleteDatasource,
  fetchDatasources,
};
