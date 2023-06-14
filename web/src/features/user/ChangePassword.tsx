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
import { Banner, Button, Form } from '@douyinfe/semi-ui';
import { UserSrv } from '@src/services';
import { ChangePassword } from '@src/types';
import React, { useRef, useState } from 'react';
import { Notification } from '@src/components';
import { isEmpty } from 'lodash-es';
import { useNavigate } from 'react-router-dom';
import { ApiKit } from '@src/utils';

const ChangePassword: React.FC = () => {
  const navigate = useNavigate();
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string>('');
  return (
    <Form
      className="linsight-form"
      onSubmit={async (values: ChangePassword) => {
        setSubmitting(true);
        try {
          await UserSrv.changePassword(values);
          setErrorMsg('');
          formApi.current.setValues({});
          Notification.success('Password change successfully');
        } catch (err) {
          setErrorMsg(ApiKit.getErrorMsg(err));
        } finally {
          setSubmitting(false);
        }
      }}
      validateFields={(values: ChangePassword) => {
        const errors: any = {};
        if (values.newPassword !== values.confirmPassword) {
          errors.confirmPassword = 'Confirm password not match new password';
        }
        if (isEmpty(errors)) {
          return null;
        }
        return errors;
      }}
      getFormApi={(api: any) => {
        formApi.current = api;
      }}>
      <Form.Section text="Change your password">
        <Form.Input
          field="oldPassword"
          label="Old passowrd"
          mode="password"
          rules={[{ required: true, message: 'Old password is requred' }]}
        />
        <Form.Input
          field="newPassword"
          label="New passowrd"
          mode="password"
          rules={[{ required: true, message: 'New password is requred' }]}
        />
        <Form.Input
          field="confirmPassword"
          label="Confirm passowrd"
          mode="password"
          rules={[{ required: true, message: 'Confirm password is requred' }]}
        />
        {!isEmpty(errorMsg) && (
          <Form.Slot>
            <Banner fullMode={false} type="danger" description={errorMsg} />
          </Form.Slot>
        )}
        <div style={{ display: 'flex', gap: 4, marginTop: 12 }}>
          <Button
            loading={submitting}
            onClick={() => {
              formApi.current.submitForm();
            }}>
            Change Password
          </Button>
          <Button
            type="tertiary"
            onClick={() => {
              navigate({ pathname: '/user/profile' });
            }}>
            Cancel
          </Button>
        </div>
      </Form.Section>
    </Form>
  );
};

export default ChangePassword;
