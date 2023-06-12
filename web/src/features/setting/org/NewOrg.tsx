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
import { Button, Card, Form, Typography } from '@douyinfe/semi-ui';
import { Org } from '@src/types';
import { Icon, Notification } from '@src/components';
import React, { useRef, useState } from 'react';
import { ApiKit } from '@src/utils';
import { OrgSrv } from '@src/services';
import { useNavigate } from 'react-router-dom';
const { Title } = Typography;
const { Meta } = Card;

const NewOrg: React.FC = () => {
  const navigate = useNavigate();
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const gotoOrgListPage = () => {
    navigate({ pathname: '/setting/orgs' });
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
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={gotoOrgListPage} underline>
                Setting
              </Title>
              <Title heading={3}>/ New Organization</Title>
            </div>
          }
          description="Each organization contains their own dashboards, data sources, and configuration etc."
          avatar={<Icon icon="org" />}
        />
      }>
      <Form
        getFormApi={(api: any) => (formApi.current = api)}
        onSubmit={async (values: Org) => {
          try {
            setSubmitting(true);
            await OrgSrv.createOrg(values);
            Notification.success('Create organization successfully!');
            gotoOrgListPage();
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input label="Organizatio name" field="name" rules={[{ required: true, message: 'Name is required' }]} />
        <div className="setting-buttons">
          <Button type="tertiary" onClick={gotoOrgListPage}>
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

export default NewOrg;
