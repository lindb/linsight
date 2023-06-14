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
import { Button, Input, Modal, Table, Typography } from '@douyinfe/semi-ui';
import { IconPlusStroked, IconSearchStroked, IconDeleteStroked } from '@douyinfe/semi-icons';
import { StatusTip, Notification } from '@src/components';
import { useRequest } from '@src/hooks';
import { OrgSrv } from '@src/services';
import React, { MutableRefObject, useRef, useState } from 'react';
import { isEmpty, debounce } from 'lodash-es';
import { createSearchParams, useNavigate } from 'react-router-dom';
import { Org, SearchOrg } from '@src/types';
import { ApiKit } from '@src/utils';

const { Text } = Typography;

const OrgList: React.FC = () => {
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);
  const [deleteVisible, setDeleteVisible] = useState(false);
  const params = useRef() as MutableRefObject<SearchOrg>;
  const currentOrg = useRef() as MutableRefObject<Org>;
  const { loading, result, error, refetch } = useRequest(['search_orgs'], () => {
    return OrgSrv.fetchOrg(params.current);
  });
  const searchOrg = debounce(refetch, 200);
  return (
    <div>
      <div style={{ display: 'flex', marginBottom: 12, gap: 4 }}>
        <Input
          prefix={<IconSearchStroked />}
          placeholder="Search organizations"
          onChange={(value: string) => {
            params.current = { name: value };
            searchOrg();
          }}
          onEnterPress={() => searchOrg()}
        />
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            navigate({ pathname: '/setting/orgs/new' });
          }}>
          New Organization
        </Button>
      </div>
      <Table
        size="small"
        bordered
        rowKey="uid"
        empty={<StatusTip isLoading={loading} isEmpty={isEmpty(result?.organizations)} error={error} />}
        dataSource={result?.organizations || []}
        columns={[
          {
            title: 'Name',
            dataIndex: 'name',
            render: (_text: any, r: Org, _index: any) => {
              return (
                <>
                  <Text
                    link
                    onClick={() => {
                      navigate({
                        pathname: '/setting/orgs/edit',
                        search: `${createSearchParams({
                          uid: `${r.uid}`,
                        })}`,
                      });
                    }}>
                    {r.name}
                  </Text>
                </>
              );
            },
          },
          {
            key: 'action',
            width: 50,
            align: 'center',
            dataIndex: 'action',
            render: (_text: any, r: Org, _index: any) => {
              return (
                <Button
                  type="danger"
                  icon={<IconDeleteStroked />}
                  onClick={() => {
                    currentOrg.current = r;
                    setDeleteVisible(true);
                  }}
                />
              );
            },
          },
        ]}
      />
      <Modal
        title={
          <div>
            Delete [<Text type="danger">{currentOrg.current?.name}</Text>] organization
          </div>
        }
        motion={false}
        visible={deleteVisible}
        onCancel={() => setDeleteVisible(false)}
        footer={
          <>
            <Button
              type="tertiary"
              onClick={() => {
                setDeleteVisible(false);
              }}>
              Cancel
            </Button>
            <Button
              type="danger"
              theme="solid"
              loading={submitting}
              onClick={async () => {
                setSubmitting(true);
                try {
                  if (currentOrg.current && currentOrg.current.uid) {
                    await OrgSrv.deleteOrg(currentOrg.current.uid);
                    setDeleteVisible(false);
                    refetch();
                    Notification.success('Organization deleted!');
                  }
                } catch (err) {
                  console.warn('delete organization error', err);
                  Notification.error(ApiKit.getErrorMsg(err));
                } finally {
                  setSubmitting(false);
                }
              }}>
              Yes
            </Button>
          </>
        }>
        Are you sure you want to remove this organization?
      </Modal>
    </div>
  );
};

export default OrgList;
