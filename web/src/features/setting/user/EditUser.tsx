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
import { Button, Card, Form, Modal, Typography } from '@douyinfe/semi-ui';
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
  const [visible, setVisible] = useState(false);
  const [searchParams] = useSearchParams();
  const uid = `${searchParams.get('uid')}`;
  const { loading, result, refetch } = useRequest(['get_user', uid], () => {
    return UserSrv.getUserByUID(uid);
  });
  const navigate = useNavigate();

  const gotoUserListPage = () => {
    navigate({ pathname: '/setting/users' });
  };

  const userIsDisable = () => {
    return get(result, 'isDisabled', false);
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
            <div className="setting-buttons">
              <Button type="primary" icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
                Save
              </Button>
              <Button
                type="tertiary"
                loading={submitting}
                onClick={async () => {
                  try {
                    setSubmitting(true);
                    if (userIsDisable()) {
                      await UserSrv.enableUserByUID(uid);
                      await refetch();
                      Notification.success('User enabled!');
                    } else {
                      setVisible(true);
                    }
                  } catch (err) {
                    Notification.error(ApiKit.getErrorMsg(err));
                  } finally {
                    setSubmitting(false);
                  }
                }}>
                {userIsDisable() ? 'Enable User' : 'Disable User'}
              </Button>
            </div>
          </Form.Slot>
        </Form.Section>
      </Form>
      <Modal
        title={<div>Disable user</div>}
        motion={false}
        visible={visible}
        onCancel={() => setVisible(false)}
        footer={
          <>
            <Button
              type="tertiary"
              onClick={() => {
                setVisible(false);
              }}>
              Cancel
            </Button>
            <Button
              type="danger"
              theme="solid"
              loading={submitting}
              onClick={async () => {
                try {
                  setSubmitting(true);
                  await UserSrv.disableUserByUID(uid);
                  await refetch();
                  Notification.success('User disabled!');
                  setVisible(false);
                } catch (err) {
                  Notification.error(ApiKit.getErrorMsg(err));
                } finally {
                  setSubmitting(false);
                }
              }}>
              Yes
            </Button>
          </>
        }>
        Are you sure you want to disable this user?
      </Modal>
    </Card>
  );
};

export default EditUser;
