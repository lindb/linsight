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
import { Button, Input, Table, Typography } from '@douyinfe/semi-ui';
import { IconPlusStroked, IconSearchStroked } from '@douyinfe/semi-icons';
import { StatusTip } from '@src/components';
import { useRequest } from '@src/hooks';
import { UserSrv } from '@src/services';
import React, { MutableRefObject, useRef } from 'react';
import { isEmpty, debounce } from 'lodash-es';
import { createSearchParams, useNavigate } from 'react-router-dom';
import { SearchUser, User } from '@src/types';

const { Text } = Typography;

const UserList: React.FC = () => {
  const navigate = useNavigate();
  const params = useRef() as MutableRefObject<SearchUser>;
  const { loading, result, error, refetch } = useRequest(['search_users'], () => {
    return UserSrv.fetchUsers(params.current);
  });
  const searchUser = debounce(refetch, 200);
  return (
    <div>
      <div style={{ display: 'flex', marginBottom: 12, gap: 4 }}>
        <Input
          prefix={<IconSearchStroked />}
          placeholder="Search users(user name/name/email)"
          onChange={(value: string) => {
            params.current = { query: value };
            searchUser();
          }}
          onEnterPress={() => searchUser()}
        />
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            navigate({ pathname: '/setting/users/new' });
          }}>
          New User
        </Button>
      </div>
      <Table
        size="small"
        bordered
        rowKey="uid"
        empty={<StatusTip isLoading={loading} isEmpty={isEmpty(result?.users)} error={error} />}
        dataSource={result?.users || []}
        columns={[
          {
            title: 'User name',
            dataIndex: 'userName',
            render: (_text: any, r: User, _index: any) => {
              return (
                <>
                  <Text
                    link
                    onClick={() => {
                      navigate({
                        pathname: '/setting/users/edit',
                        search: `${createSearchParams({
                          uid: `${r.uid}`,
                        })}`,
                      });
                    }}>
                    {r.userName}
                  </Text>
                </>
              );
            },
          },
          { title: 'Name', dataIndex: 'name' },
          { title: 'Email', dataIndex: 'email' },
        ]}
      />
    </div>
  );
};

export default UserList;
