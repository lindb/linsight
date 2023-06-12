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
import { Button, Card, Form, TabPane, Tabs, Tag, Typography } from '@douyinfe/semi-ui';
import { IconCandlestickChartStroked } from '@douyinfe/semi-icons';
import { Icon, Loading, Notification } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { TeamSrv } from '@src/services';
import { Team } from '@src/types';
import React, { useContext, useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { get } from 'lodash-es';
import { ApiKit } from '@src/utils';
import { useRequest } from '@src/hooks';
const { Meta } = Card;
const { Title, Text } = Typography;

const EditTeam: React.FC = () => {
  const { boot } = useContext(PlatformContext);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const teamUID = `${searchParams.get('uid')}`;
  const location = useLocation();
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const { loading, result } = useRequest(['get_team', teamUID], () => TeamSrv.GetTeamByUID(teamUID));
  const gotoTeamListPage = () => {
    navigate({ pathname: '/setting/org/teams' });
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
      <Tabs defaultActiveKey={location.pathname}>
        <TabPane itemKey="/setting/org/teams/edit/setting" tab="Setting" icon={<IconCandlestickChartStroked />}>
          <Form
            getFormApi={(api: any) => (formApi.current = api)}
            onSubmit={async (values: Team) => {
              try {
                setSubmitting(true);
                values.uid = teamUID;
                await TeamSrv.updateTeam(values);
                Notification.success('Update team successfully!');
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
                Update
              </Button>
            </div>
          </Form>
        </TabPane>
      </Tabs>
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
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={() => gotoTeamListPage()} underline>
                Teams
              </Title>
              <Title heading={3}>/ {get(result, 'name', 'N/A')}</Title>
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
      {renderContent()}
    </Card>
  );
};

export default EditTeam;
