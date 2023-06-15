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
import React, { useState } from 'react';
import { Banner, Button, Card, Form, Typography } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { UserSrv } from '@src/services';
import { Icon, Notification } from '@src/components';
import { useNavigate } from 'react-router-dom';
import { User } from '@src/types';
import { ApiKit } from '@src/utils';
import { isEmpty } from 'lodash-es';

const { Meta } = Card;
const { Title, Text } = Typography;

const NewUser: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const navigate = useNavigate();

  const gotoUserListPage = () => {
    navigate({ pathname: '/setting/users' });
  };

  return (
    <Card
      className="setting-page"
      bordered={false}
      bodyStyle={{ padding: 24 }}
      title={
        <Meta
          className="setting-meta"
          title={
            <div className="meta-title">
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={() => gotoUserListPage()} underline>
                Setting
              </Title>
              <Title heading={3}>/ New User</Title>
            </div>
          }
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Create a new Linsight user</Text>
            </div>
          }
          avatar={<Icon icon="user" />}
        />
      }>
      <Form
        className="linsight-form setting-form"
        onSubmit={async (values: User) => {
          try {
            setSubmitting(true);
            await UserSrv.createUser(values);
            setErrorMsg('');
            Notification.success('Create user successfully');
            gotoUserListPage();
          } catch (err) {
            setErrorMsg(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input field="userName" label="Username" rules={[{ required: true, message: 'Password is required' }]} />
        <Form.Input field="name" label="Name" />
        <Form.Input field="email" label="Email" rules={[{ required: true, message: 'Password is required' }]} />
        <Form.Input
          field="password"
          label="Password"
          mode="password"
          rules={[{ required: true, message: 'Password is required' }]}
        />
        {!isEmpty(errorMsg) && (
          <Form.Slot>
            <Banner fullMode={false} type="danger" description={errorMsg} />
          </Form.Slot>
        )}
        <Form.Slot>
          <div className="setting-buttons">
            <Button type="tertiary" onClick={gotoUserListPage}>
              Back
            </Button>
            <Button type="primary" icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
              Create User
            </Button>
          </div>
        </Form.Slot>
      </Form>
    </Card>
  );
};

export default NewUser;
