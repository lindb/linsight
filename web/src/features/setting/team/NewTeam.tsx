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
import { Button, Card, Form, Tag, Typography } from '@douyinfe/semi-ui';
import { Team } from '@src/types';
import { Icon, Notification } from '@src/components';
import React, { useContext, useRef, useState } from 'react';
import { ApiKit } from '@src/utils';
import { TeamSrv } from '@src/services';
import { get } from 'lodash-es';
import { PlatformContext } from '@src/contexts';
import { useNavigate } from 'react-router-dom';
const { Title, Text } = Typography;
const { Meta } = Card;

const NewTeam: React.FC = () => {
  const { boot } = useContext(PlatformContext);
  const navigate = useNavigate();
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const gotoTeamListPage = () => {
    navigate({ pathname: '/setting/org/teams' });
  };
  return (
    <Card
      className="setting-page"
      bordered={false}
      bodyStyle={{ padding: 12 }}
      title={
        <Meta
          className="setting-meta"
          title={
            <div className="meta-title">
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={() => gotoTeamListPage()} underline>
                Setting
              </Title>
              <Title heading={3}>/ New Team</Title>
            </div>
          }
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Current organization:</Text>
              <Tag>{get(boot, 'user.org.name', 'N/A')}</Tag>
            </div>
          }
          avatar={<Icon icon="team" />}
        />
      }>
      <Form
        getFormApi={(api: any) => (formApi.current = api)}
        onSubmit={async (values: Team) => {
          try {
            setSubmitting(true);
            await TeamSrv.createTeam(values);
            Notification.success('Create team successfully!');
            gotoTeamListPage();
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input label="Name" field="name" rules={[{ required: true, message: 'Name is required' }]} />
        <Form.Input label="Email" field="email" />
        <div className="setting-buttons">
          <Button type="tertiary" onClick={gotoTeamListPage}>
            Back
          </Button>
          <Button
            loading={submitting}
            onClick={() => {
              formApi.current.submitForm();
            }}>
            Create
          </Button>
        </div>
      </Form>
    </Card>
  );
};

export default NewTeam;
