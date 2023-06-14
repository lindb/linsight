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
import { Button, Card, Form, Typography } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { UserSrv } from '@src/services';
import { Icon, Notification } from '@src/components';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useRequest } from '@src/hooks';
import { get } from 'lodash-es';
import { User } from '@src/types';
import { ApiKit } from '@src/utils';
const { Meta } = Card;
const { Title, Text } = Typography;

const EditUser: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const [searchParams] = useSearchParams();
  const uid = `${searchParams.get('uid')}`;
  const { loading, result } = useRequest(['get_user', uid], () => {
    return UserSrv.getUserByUID(uid);
  });

  const navigate = useNavigate();

  const gotoUserListPage = () => {
    navigate({ pathname: '/setting/users' });
  };

  return (
    <Card
      className="setting-page"
      loading={loading}
      bordered={false}
      bodyStyle={{ padding: 24 }}
      title={
        <Meta
          className="setting-meta"
          title={
            <div className="meta-title">
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={() => gotoUserListPage()} underline>
                Users
              </Title>
              <Title heading={3}>/ {get(result, 'userName', 'N/A')}</Title>
            </div>
          }
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Setting for current user</Text>
            </div>
          }
          avatar={<Icon icon="user" />}
        />
      }>
      <Form
        className="linsight-form setting-form"
        initValues={result || {}}
        allowEmpty
        onSubmit={async (values: User) => {
          try {
            setSubmitting(true);
            values.uid = uid;
            await UserSrv.updateUser(values);
            Notification.success('Update user successfully!');
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Section text="Basic Information">
          <Form.Input field="userName" label="Username" rules={[{ required: true, message: 'Password is required' }]} />
          <Form.Input field="name" label="Name" />
          <Form.Input field="email" label="Email" rules={[{ required: true, message: 'Password is required' }]} />
          <Form.Slot>
            <Button type="primary" icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
              Save
            </Button>
          </Form.Slot>
        </Form.Section>
      </Form>
    </Card>
  );
};

export default EditUser;
