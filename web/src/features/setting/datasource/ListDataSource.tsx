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
import React, { useContext, useState } from 'react';
import { Button, Divider, Input, List, Tag, Typography } from '@douyinfe/semi-ui';
import { IconSearchStroked, IconPlusStroked } from '@douyinfe/semi-icons';
import { DatasourceSrv } from '@src/services';
import { DatasourceRepositoryInst, DatasourceSetting } from '@src/types';
import { isEmpty, filter, includes } from 'lodash-es';
import { createSearchParams, useNavigate } from 'react-router-dom';
import { Icon, StatusTip } from '@src/components';
import { useRequest } from '@src/hooks';
import DeleteDatasourceButton from './components/DeleteDatasourceButton';
import { PlatformContext } from '@src/contexts';
import { DatasourceKit, TimeKit } from '@src/utils';

const { Title, Text } = Typography;

const ListDataSource: React.FC = () => {
  const { theme } = useContext(PlatformContext);
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const { result, loading, error, refetch } = useRequest(['list_datasources'], () => DatasourceSrv.fetchDatasources());
  return (
    <div>
      <div style={{ display: 'flex', gap: 8, marginBottom: 20 }}>
        <Input
          prefix={<IconSearchStroked />}
          placeholder="Filter datasources(name/url/type)"
          showClear
          onChange={(v: string) => {
            setSearch(v);
          }}
        />
        <Button icon={<IconPlusStroked />} onClick={() => navigate('/setting/datasources/new')}>
          New
        </Button>
      </div>
      <List
        size="small"
        bordered
        dataSource={
          isEmpty(search)
            ? result
            : filter(result, (ds: DatasourceSetting) => {
                return includes(ds.name, search) || includes(ds.url, search) || includes(ds.type, search);
              })
        }
        emptyContent={<StatusTip isLoading={loading} isEmpty={isEmpty(result)} error={error} />}
        renderItem={(ds: DatasourceSetting) => {
          const item = DatasourceRepositoryInst.get(ds.type);
          return (
            <List.Item
              key={ds.uid}
              header={<img src={`${item?.getLogo(theme)}`} width={48} />}
              extra={
                <div className="button-group">
                  <Button
                    type="tertiary"
                    icon={<Icon icon="dashboard" />}
                    onClick={() => {
                      navigate('/dashboard');
                    }}>
                    New dashboard
                  </Button>
                  <Button
                    type="tertiary"
                    icon={<Icon icon="explore" />}
                    onClick={() => {
                      navigate({ pathname: '/explore', search: DatasourceKit.getDatasourceDefaultParams(ds.uid) });
                    }}>
                    Explore
                  </Button>
                  <DeleteDatasourceButton
                    uid={ds.uid}
                    name={ds.name}
                    onCompleted={() => {
                      refetch();
                    }}
                  />
                </div>
              }
              main={
                <div
                  style={{ cursor: 'pointer', width: '100%' }}
                  onClick={() =>
                    navigate({
                      pathname: '/setting/datasources/edit',
                      search: `${createSearchParams({
                        uid: ds.uid,
                      })}`,
                    })
                  }>
                  <Title heading={5}>
                    {ds.name}
                    {ds.isDefault && (
                      <Tag color="orange" style={{ marginLeft: 8 }}>
                        Default
                      </Tag>
                    )}
                  </Title>
                  <div style={{ marginTop: 8, display: 'flex', alignItems: 'center' }}>
                    <Text>{item?.Name}</Text>
                    <Divider layout="vertical" style={{ margin: '0 8px' }} />
                    <Text>{ds.url}</Text>
                    {ds.timeZone && (
                      <>
                        <Divider layout="vertical" style={{ margin: '0 8px' }} />
                        <Text>{ds.timeZone}</Text>
                        <Tag style={{ marginLeft: 6 }}>{TimeKit.utcOffset(ds.timeZone)}</Tag>
                      </>
                    )}
                  </div>
                </div>
              }
            />
          );
        }}
      />
    </div>
  );
};

export default ListDataSource;
