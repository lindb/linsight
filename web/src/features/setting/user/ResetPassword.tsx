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
import React, { useRef, useState } from 'react';
import { Notification } from '@src/components';
import { ResetPassword as ResetPWD } from '@src/types';
import { ApiKit } from '@src/utils';
import { UserSrv } from '@src/services';

const ResetPassword: React.FC<{ userUid: string; visible: boolean; setVisible: (v: boolean) => void }> = (props) => {
  const { userUid, visible, setVisible } = props;
  const [submitting, setSubmitting] = useState(false);
  const formApi = useRef<any>();
  return (
    <Modal
      title="Reset password"
      visible={visible}
      closeOnEsc
      motion={false}
      onCancel={() => setVisible(false)}
      okText="Reset Password"
      confirmLoading={submitting}
      onOk={() => {
        formApi.current.submitForm();
      }}>
      <Form
        getFormApi={(api: any) => {
          formApi.current = api;
        }}
        onSubmit={async (values: ResetPWD) => {
          try {
            setSubmitting(true);
            values.userUid = userUid;
            await UserSrv.resetPassword(values);
            Notification.success('Password reseted!');
            setVisible(false);
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input
          field="password"
          label="Password"
          mode="password"
          rules={[{ required: true, message: 'Password is required' }]}
        />
      </Form>
    </Modal>
  );
};

export default ResetPassword;
