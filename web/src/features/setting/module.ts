import { Feature, FeatureRepositoryInst } from '@src/types';
import EditUser from '@src/features/setting/user/EditUser';
import ListDataSource from '@src/features/setting/datasource/ListDataSource';
import EditDataSource from '@src/features/setting/datasource/EditDataSource';

FeatureRepositoryInst.register(new Feature('/setting/user/edit', EditUser))
  .register(new Feature('/setting/datasources', ListDataSource))
  .register(new Feature('/setting/datasource', EditDataSource));
