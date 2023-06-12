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
import { Icon, Loading, Notification } from '@src/components';
import { OrgSrv } from '@src/services';
import { Org } from '@src/types';
import React, { useEffect, useRef, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { get } from 'lodash-es';
import { ApiKit } from '@src/utils';
import { useRequest } from '@src/hooks';
const { Meta } = Card;
const { Title } = Typography;

const EditOrg: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const orgUID = `${searchParams.get('uid')}`;
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const { loading, result } = useRequest(['get_org', orgUID], () => OrgSrv.GetOrgByUID(orgUID));
  const gotoOrgListPage = () => {
    navigate({ pathname: '/setting/orgs' });
  };
  useEffect(() => {
    if (result) {
      formApi.current.setValues(result);
    }
  }, [result]);
  const renderContent = () => {
    if (loading) {
      return (
        <div style={{ width: '100%', height: 400, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Loading />
        </div>
      );
    }
    return (
      <Form
        getFormApi={(api: any) => (formApi.current = api)}
        onSubmit={async (values: Org) => {
          try {
            setSubmitting(true);
            values.uid = orgUID;
            await OrgSrv.updateOrg(values);
            Notification.success('Update organization successfully!');
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input label="Organization name" field="name" rules={[{ required: true, message: 'Name is required' }]} />
        <div className="setting-buttons">
          <Button type="tertiary" onClick={gotoOrgListPage}>
            Back
          </Button>
          <Button
            loading={submitting}
            onClick={() => {
              formApi.current.submitForm();
            }}>
            Update
          </Button>
        </div>
      </Form>
    );
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
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={() => gotoOrgListPage()} underline>
                Organizations
              </Title>
              <Title heading={3}>/ {get(result, 'name', 'N/A')}</Title>
            </div>
          }
          description="Each organization contains their own dashboards, data sources, and configuration etc."
          avatar={<Icon icon="org" />}
        />
      }>
      {renderContent()}
    </Card>
  );
};

export default EditOrg;
