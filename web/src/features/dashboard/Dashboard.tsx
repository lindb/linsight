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
import { DashboardSrv } from '@src/services';
import { createSearchParams, Route, Routes, useLocation, useNavigate, useSearchParams } from 'react-router-dom';
import { isEmpty, endsWith } from 'lodash-es';
import { Layout, Typography, Button, SideSheet, Form, Space } from '@douyinfe/semi-ui';
import { IconGridStroked, IconSaveStroked, IconStar, IconSettingStroked, IconStarStroked } from '@douyinfe/semi-icons';
import { DashboardStore } from '@src/stores';
import { Icon, Loading, Notification, TimePicker } from '@src/components';
import Setting from './Setting';
import PanelEditor from './PanelEditor';
import View from './View';
import { ApiKit, VariableKit } from '@src/utils';
import { observer } from 'mobx-react-lite';
import ViewPanel from './ViewPanel';
import './dashboard.scss';
import { useRequest } from '@src/hooks';

const { Header, Content } = Layout;
const { Title } = Typography;

const DashboardStar = observer(() => {
  const { dashboard } = DashboardStore;

  if (!dashboard) {
    return null;
  }

  if (dashboard.isStarred) {
    return (
      <IconStar
        size="large"
        style={{ cursor: 'pointer' }}
        onClick={() => {
          try {
            DashboardSrv.unstarDashboard(`${dashboard.uid}`);
            DashboardStore.updateDashboardProps({ isStarred: false });
            Notification.success('Unstar dashboard successfully');
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          }
        }}
      />
    );
  }
  return (
    <IconStarStroked
      size="large"
      style={{ cursor: 'pointer' }}
      onClick={() => {
        try {
          DashboardSrv.starDashboard(`${dashboard.uid}`);
          DashboardStore.updateDashboardProps({ isStarred: true });
          Notification.success('Star dashboard successfully');
        } catch (err) {
          Notification.error(ApiKit.getErrorMsg(err));
        }
      }}
    />
  );
});

const SaveDashboard: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const dashboardId = searchParams.get('d');
  const [visible, setVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const saveDashboard = async (values: any) => {
    try {
      setSubmitting(true);
      DashboardStore.updateDashboardProps({ title: values.title });
      if (values.saveVariable) {
        // save current variable selected values
        VariableKit.setVariableValues(searchParams, DashboardStore.getVariables());
      }
      const success = await DashboardStore.saveDashboard();
      if (success && !dashboardId) {
        // if create dashboard successfully, need set uid to url params
        searchParams.set('d', `${DashboardStore.dashboard.uid}`);
        setSearchParams(searchParams);
      }
      setVisible(false);
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <>
      <Button
        type="tertiary"
        icon={<IconSaveStroked />}
        onClick={async () => {
          setVisible(true);
        }}
      />
      <SideSheet
        size="medium"
        motion={false}
        closeOnEsc
        visible={visible}
        onCancel={() => setVisible(false)}
        headerStyle={{ borderBottom: '1px solid var(--semi-color-border)' }}
        bodyStyle={{ borderBottom: '1px solid var(--semi-color-border)' }}
        title={
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <IconGridStroked size="extra-large" />
            <Typography.Title heading={4}>Save dashboard</Typography.Title>
          </div>
        }>
        <Form
          className="linsight-form"
          initValues={DashboardStore.dashboard}
          onSubmit={(values: any) => saveDashboard(values)}>
          {({ formApi }) => (
            <>
              {isEmpty(dashboardId) && (
                <Form.Input
                  field="title"
                  label="Dashboard title"
                  rules={[{ required: true, message: 'Dashboard title is required' }]}
                />
              )}
              <Form.Checkbox field="saveVariable" noLabel>
                Save current variable values
              </Form.Checkbox>
              <Form.TextArea field="comment" placeholder="Add a comment to describe your changes" noLabel />
              <Space style={{ marginTop: 12 }}>
                <Button type="tertiary" onClick={() => setVisible(false)}>
                  Cancel
                </Button>
                <Button icon={<IconSaveStroked />} loading={submitting} onClick={() => formApi.submitForm()}>
                  Save
                </Button>
              </Space>
            </>
          )}
        </Form>
      </SideSheet>
    </>
  );
};

const Dashboard: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const location = useLocation();
  const dashboardId = searchParams.get('d');
  const dashboard = DashboardStore.dashboard;
  const { loading } = useRequest(
    ['load-dashboard', dashboardId],
    async () => {
      return DashboardStore.loadDashbaord(dashboardId);
    },
    {}
  );

  const isEditDashboard = (): boolean => {
    return (
      endsWith(location.pathname, '/edit') ||
      endsWith(location.pathname, '/setting') ||
      endsWith(location.pathname, '/panel/view')
    );
  };

  if (loading) {
    return (
      <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Loading />
      </div>
    );
  }

  if (!dashboard) {
    // FIXME: use error page
    return <div>Dashboard not exit</div>;
  }

  return (
    <Layout>
      <Header className="linsight-feature-header dashboard">
        <div className="title">
          <IconGridStroked size="large" />
          <Title heading={6}>{DashboardStore.dashboard?.title}</Title>
          <DashboardStar />
        </div>
        <div style={{ marginRight: 12, gap: 4, display: 'flex' }}>
          {isEditDashboard() && (
            <Button
              type="tertiary"
              icon={<Icon icon="back2" />}
              onClick={() => {
                navigate({
                  pathname: '/dashboard',
                  search: dashboard.uid ? `${createSearchParams({ d: dashboard.uid })}` : '',
                });
              }}
            />
          )}
          <Button
            type="tertiary"
            icon={<Icon icon="panel-add" />}
            onClick={() => {
              DashboardStore.addPanel(DashboardStore.createPanelConfig());
            }}
          />
          <SaveDashboard />
          <Button
            type="tertiary"
            icon={<IconSettingStroked />}
            onClick={() => {
              navigate({
                pathname: '/dashboard/setting',
                search: dashboard.uid ? `${createSearchParams({ d: dashboard.uid })}` : '',
              });
            }}
          />
          <TimePicker />
        </div>
      </Header>
      <Content>
        <Routes>
          <Route path="/setting" element={<Setting />} />
          <Route path="/panel/edit" element={<PanelEditor />} />
          <Route path="/panel/view" element={<ViewPanel />} />
          <Route path="/" element={<View />} />
        </Routes>
      </Content>
    </Layout>
  );
};

export default observer(Dashboard);
