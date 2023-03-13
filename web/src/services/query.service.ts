import { ApiPath } from '@src/constants';
import { DataQuery } from '@src/types';
import { ApiKit } from '@src/utils';

const query = (req: DataQuery): Promise<any> => {
  return ApiKit.PUT<any>(ApiPath.DataQuery, req);
};

export default { query };
