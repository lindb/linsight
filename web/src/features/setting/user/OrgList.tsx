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
import { Button, Form, Select, Table } from '@douyinfe/semi-ui';
import { useRequest } from '@src/hooks';
import { UserSrv } from '@src/services';
import { RoleList, UserOrg } from '@src/types';
import { Notification } from '@src/components';
import React, { useState } from 'react';
import AddOrgModal from './AddOrgModal';
import { ApiKit } from '@src/utils';

const OrgList: React.FC<{ userUid: string }> = (props) => {
  const { userUid } = props;
  const {
    result: orgList,
    refetch,
    loading,
  } = useRequest(['get_org_list_for_user', userUid], () => {
    return UserSrv.getOrgListForUser(userUid);
  });
  const [addOrgVisible, setAddOrgVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  return (
    <>
      <Form className="linsight-form setting-form" style={{ marginTop: 24 }}>
        <Form.Section text="Organizations">
          <Table
            bordered
            showHeader={false}
            size="small"
            rowKey="orgUid"
            loading={loading}
            dataSource={orgList || []}
            pagination={false}
            columns={[
              {
                title: 'Org. name',
                dataIndex: 'orgName',
              },
              {
                title: 'Role',
                dataIndex: 'role',
                render: (_text: string, r: UserOrg, _index: number) => {
                  return (
                    <Select
                      defaultValue={r.role}
                      optionList={RoleList}
                      onChange={(value: any) => {
                        r.role = value;
                        UserSrv.updateUserOrg(userUid, r)
                          .then(() => {
                            refetch();
                            Notification.success('Role changed!');
                          })
                          .catch((err) => {
                            Notification.error(ApiKit.getErrorMsg(err));
                          });
                      }}
                    />
                  );
                },
              },
              {
                render: (_text: string, r: UserOrg, _index: number) => {
                  return (
                    <Button
                      type="danger"
                      loading={submitting}
                      onClick={async () => {
                        setSubmitting(true);
                        try {
                          await UserSrv.deleteUserOrg(userUid, r.orgUid);
                          Notification.success('Remove organization successfully!');
                          refetch();
                        } catch (err) {
                          Notification.error(ApiKit.getErrorMsg(err));
                        } finally {
                          setSubmitting(false);
                        }
                      }}>
                      Remove
                    </Button>
                  );
                },
              },
            ]}
          />
          <Form.Slot>
            <Button
              type="tertiary"
              onClick={() => {
                setAddOrgVisible(true);
              }}>
              Add to organization
            </Button>
          </Form.Slot>
        </Form.Section>
      </Form>
      {addOrgVisible && (
        <AddOrgModal
          orgList={orgList}
          visible={addOrgVisible}
          setVisible={(v: boolean) => {
            setAddOrgVisible(v);
            refetch();
          }}
          userUid={userUid}
        />
      )}
    </>
  );
};

export default OrgList;
