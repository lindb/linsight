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
import { Button, Form, Input, Modal, Table, Tag, Typography, Select } from '@douyinfe/semi-ui';
import { IconDeleteStroked, IconSearchStroked, IconPlusStroked } from '@douyinfe/semi-icons';
import { StatusTip, Notification } from '@src/components';
import { useRequest } from '@src/hooks';
import { OrgSrv, TeamSrv } from '@src/services';
import { AddTeamMembers, OrgUser, PermissionList, SearchTeamMember, TeamMember } from '@src/types';
import React, { useRef, useState } from 'react';
import { debounce, isEmpty } from 'lodash-es';
import { ApiKit } from '@src/utils';

const { Text } = Typography;

const AddMembers: React.FC<{ teamUid: string; visible: boolean; setVisible: (v: boolean) => void }> = (props) => {
  const { teamUid, visible, setVisible } = props;
  const prefix = useRef<string>('');
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const { result, refetch, loading } = useRequest(['search_org_members'], () => {
    return OrgSrv.getUsersForCurrentOrg({ prefix: prefix.current });
  });

  const renderMultipleWithCustomTag = (optionNode: any, mProps: { onClose: any }) => {
    const { onClose } = mProps;
    const content = (
      <Tag avatarSrc={optionNode.avatar} avatarShape="square" closable={true} onClose={onClose} size="large">
        {optionNode.userName}({optionNode.name})
      </Tag>
    );
    return {
      isRenderInTag: false,
      content,
    };
  };
  const searchUser = debounce(refetch, 300);

  return (
    <Modal
      visible={visible}
      size="large"
      closeOnEsc
      maskClosable={false}
      onCancel={() => setVisible(false)}
      motion={false}
      onOk={() => {
        formApi.current.submitForm();
      }}
      confirmLoading={submitting}
      okText="Add user to team"
      title="Add user to team">
      <Form
        className="linsight-form"
        getFormApi={(api: any) => {
          formApi.current = api;
        }}
        onSubmit={async (values: AddTeamMembers) => {
          try {
            setSubmitting(true);
            await TeamSrv.addTeamMembers(teamUid, values);
            Notification.success('Add team members successfully!');
            setVisible(false);
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Select
          field="userUids"
          label="User"
          loading={loading}
          placeholder="Please select user"
          onSearch={(value: string) => {
            prefix.current = value;
            searchUser();
          }}
          filter
          remote
          multiple
          style={{ width: '100%' }}
          renderSelectedItem={renderMultipleWithCustomTag}
          rules={[{ required: true, message: 'User is required' }]}>
          {(result || []).map((user: OrgUser) => {
            return (
              <Select.Option key={user.userUid} showTick={false} value={user.userUid} {...user}>
                <div style={{ display: 'flex', flexDirection: 'column' }}>
                  <Text>
                    {user.name}({user.userName})
                  </Text>
                  <Text size="small" type="tertiary">
                    {user.email}
                  </Text>
                </div>
              </Select.Option>
            );
          })}
        </Form.Select>
        <Form.Select
          field="permission"
          label="Permission"
          placeholder="Please select permission"
          optionList={PermissionList}
          rules={[{ required: true, message: 'Permission is required' }]}
        />
      </Form>
    </Modal>
  );
};

const MemberList: React.FC<{ teamUid: string }> = (props) => {
  const { teamUid } = props;
  const params = useRef<SearchTeamMember>({});
  const { loading, result, error, refetch } = useRequest(['search_team_members'], () => {
    return TeamSrv.getTeamMembers(teamUid, params.current);
  });
  const [visible, setVisible] = useState(false);
  const searchMember = debounce(refetch, 300);
  return (
    <div>
      {visible && (
        <AddMembers
          teamUid={teamUid}
          visible={visible}
          setVisible={(v: boolean) => {
            setVisible(v);
            refetch();
          }}
        />
      )}
      <div style={{ display: 'flex', marginBottom: 12, gap: 4 }}>
        <Input
          prefix={<IconSearchStroked />}
          placeholder="Search members"
          onChange={(value: string) => {
            params.current.user = value;
            searchMember();
          }}
          showClear
          onEnterPress={() => searchMember()}
        />
        <Select
          style={{ width: 250 }}
          placeholder="Permission"
          optionList={PermissionList}
          multiple
          onChange={(value: any) => {
            params.current.permissions = value;
            searchMember();
          }}
        />
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            setVisible(true);
          }}>
          New Member
        </Button>
      </div>
      <Table
        size="small"
        bordered
        rowKey="userUid"
        empty={<StatusTip isLoading={loading} isEmpty={isEmpty(result?.members)} error={error} />}
        dataSource={result?.members || []}
        columns={[
          { title: 'User name', dataIndex: 'userName' },
          { title: 'Name', dataIndex: 'name' },
          {
            title: 'Permission',
            dataIndex: 'permission',
            render: (_text: string, r: TeamMember, _index: number) => {
              return (
                <Select
                  value={r.permission}
                  optionList={PermissionList}
                  onChange={(value: any) => {
                    TeamSrv.updateTeamMemeber(teamUid, {
                      userUid: r.userUid,
                      permission: value,
                    })
                      .then(() => {
                        refetch();
                        Notification.success('Permission changed!');
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
            width: 60,
            render: (_text: string, r: TeamMember, _index: number) => {
              return (
                <Button
                  icon={<IconDeleteStroked />}
                  onClick={async () => {
                    try {
                      await TeamSrv.removeTeamMembers(teamUid, { userUids: [r.userUid] });
                      refetch();
                      Notification.success('Member removed!');
                    } catch (err) {
                      Notification.error(ApiKit.getErrorMsg(err));
                    }
                  }}
                />
              );
            },
          },
        ]}
      />
    </div>
  );
};

export default MemberList;
