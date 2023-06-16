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
import { Form, Modal } from '@douyinfe/semi-ui';
import { useRequest } from '@src/hooks';
import { OrgSrv, UserSrv } from '@src/services';
import { Notification } from '@src/components';
import { Org, RoleList, UserOrg } from '@src/types';
import React, { useEffect, useRef, useState } from 'react';
import { find } from 'lodash-es';
import { ApiKit } from '@src/utils';

const AddOrgModal: React.FC<{
  orgList: UserOrg[];
  userUid: string;
  visible: boolean;
  setVisible: (v: boolean) => void;
}> = (props) => {
  const { orgList, visible, setVisible, userUid } = props;
  const { result } = useRequest(['get_orgs_for_user'], () => {
    return OrgSrv.getOrgsForCurrentUser();
  });
  const [orgOptions, setOrgOptions] = useState<any[]>([]);
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    const rs: any[] = [];
    (result || []).forEach((o: Org) => {
      if (!find(orgList, { orgUid: o.uid })) {
        rs.push({
          label: o.name,
          value: o.uid,
          showTick: false,
        });
      }
    });
    setOrgOptions(rs);
  }, [orgList, result]);

  return (
    <Modal
      title="Add to organization"
      motion={false}
      closeOnEsc
      visible={visible}
      okText="Add to organization"
      confirmLoading={submitting}
      onOk={() => {
        formApi.current.submitForm();
      }}
      onCancel={() => setVisible(false)}>
      <Form
        className="linsight-form setting-form"
        getFormApi={(api: any) => {
          formApi.current = api;
        }}
        onSubmit={async (values: UserOrg) => {
          try {
            setSubmitting(true);
            await UserSrv.createUserOrg(userUid, values);
            Notification.success('Join organization successfully!');
            setVisible(false);
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Select
          field="orgUid"
          label="Organization"
          style={{ width: '100%' }}
          rules={[{ required: true, message: 'Organization is required' }]}
          optionList={orgOptions}
        />
        <Form.Select
          field="role"
          label="Role"
          rules={[{ required: true, message: 'Role is required' }]}
          style={{ width: '100%' }}
          optionList={RoleList}
        />
      </Form>
    </Modal>
  );
};

export default AddOrgModal;
