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
