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
import React, { MutableRefObject, useCallback, useRef, useState } from 'react';
import { DashboardSrv } from '@src/services';
import {
  Route,
  Routes,
  useBeforeUnload,
  useLocation,
  useNavigate,
  useSearchParams,
  unstable_useBlocker as useBlocker,
} from 'react-router-dom';
import { isEmpty, startsWith, endsWith } from 'lodash-es';
import { Layout, Typography, Button, SideSheet, Form, Space, Modal, Empty } from '@douyinfe/semi-ui';
import {
  IconGridStroked,
  IconSaveStroked,
  IconPlusStroked,
  IconStar,
  IconSettingStroked,
  IconStarStroked,
} from '@douyinfe/semi-icons';
import { DashboardStore } from '@src/stores';
import { ErrorPage, Icon, Loading, Notification, TimePicker } from '@src/components';
import Setting from './Setting';
import PanelEditor from './PanelEditor';
import View from './View';
import { ApiKit, VariableKit } from '@src/utils';
import { observer } from 'mobx-react-lite';
import ViewPanel from './ViewPanel';
import './dashboard.scss';
import { useRequest } from '@src/hooks';
import { VariableContextProvider } from '@src/contexts';
import NotFoundImg from '@src/images/4042.svg';
import DashboardSearchModal from './components/DashboardSearchModal';

const { Header, Content } = Layout;
const { Title, Text } = Typography;

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
  const [confirm, setConfirm] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const blocking = useRef(true);
  const nextPath = useRef('');

  /**
   * Track dashboard if changed when goto other page
   */
  useBlocker(
    useCallback((transition: any) => {
      const nextPathName = transition.nextLocation.pathname;
      if (startsWith(nextPathName, '/dashboard')) {
        return false;
      }
      if (!DashboardStore.isDashboardChanged()) {
        // no changes unblocking
        return false;
      }
      nextPath.current = transition.nextLocation.pathname;
      if (blocking.current) {
        setConfirm(true);
      }
      return blocking.current;
    }, [])
  );

  /**
   * Track dashboard if changed when close/reflesh page
   */
  useBeforeUnload(
    useCallback((event: BeforeUnloadEvent) => {
      // TODO: handle ignore dashboard change?
      if (DashboardStore.isDashboardChanged()) {
        event.preventDefault();
        // chrome
        event.returnValue = '';
      }
    }, [])
  );

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
      if (!isEmpty(nextPath.current)) {
        navigate(nextPath.current);
      }
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
      <Modal
        title="Unsaved changes"
        visible={confirm}
        closeOnEsc
        onCancel={() => setConfirm(false)}
        footer={
          <div>
            <Button type="tertiary" onClick={() => setConfirm(false)}>
              Cancel
            </Button>
            <Button
              type="danger"
              theme="solid"
              onClick={() => {
                blocking.current = false;
                setConfirm(false);
                navigate(nextPath.current);
              }}>
              Discard
            </Button>
            <Button
              type="primary"
              onClick={() => {
                setConfirm(false);
                setVisible(true);
              }}>
              Save dashboard
            </Button>
          </div>
        }>
        Do you want to save your changes?
      </Modal>
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
  const searchRef = useRef() as MutableRefObject<any>;
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

  if (!dashboard || isEmpty(dashboard)) {
    return (
      <Layout>
        <Header className="linsight-feature-header linsight-dashboard">
          <div className="title">
            <IconGridStroked size="large" />
            <Title heading={6} className="name">
              Not found
            </Title>
          </div>
        </Header>
        <Content>
          <Empty
            title="Oops!"
            style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}
            image={<img src={NotFoundImg} style={{ width: 250, height: 250 }} />}
            darkModeImage={<img src={NotFoundImg} style={{ width: 250, height: 250 }} />}
            layout="horizontal"
            description={<Text type="danger">Dashboard not exist</Text>}>
            <div style={{ display: 'flex', gap: 6 }}>
              <Button
                type="tertiary"
                onClick={() => {
                  navigate({ pathname: '/dashboards' });
                }}>
                Back
              </Button>
              <Button
                icon={<IconPlusStroked />}
                onClick={() => {
                  navigate({ pathname: '/dashboard' });
                }}>
                New Dashboard
              </Button>
            </div>
          </Empty>
        </Content>
      </Layout>
    );
  }

  return (
    <>
      <DashboardSearchModal ref={searchRef} />
      <Layout>
        <Header className="linsight-feature-header linsight-dashboard">
          <div className="title">
            <IconGridStroked size="large" />
            <Title
              heading={6}
              className="name"
              onClick={() => {
                searchRef.current.toggleSearchModal();
              }}>
              {DashboardStore.dashboard?.title}
            </Title>
            <DashboardStar />
          </div>
          <div style={{ marginRight: 12, gap: 4, display: 'flex' }}>
            {isEditDashboard() && (
              <Button
                type="tertiary"
                icon={<Icon icon="back2" />}
                onClick={() => {
                  searchParams.delete('panel');
                  searchParams.delete('tab');
                  searchParams.delete('edit');
                  navigate({
                    pathname: '/dashboard',
                    search: searchParams.toString(),
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
                searchParams.delete('panel');
                navigate({
                  pathname: '/dashboard/setting',
                  search: searchParams.toString(),
                });
              }}
            />
            <TimePicker />
          </div>
        </Header>
        <Content>
          <VariableContextProvider variables={DashboardStore.getVariables()}>
            <Routes>
              <Route path="/setting" element={<Setting />} errorElement={<ErrorPage />} />
              <Route path="/panel/edit" element={<PanelEditor />} errorElement={<ErrorPage />} />
              <Route path="/panel/view" element={<ViewPanel />} errorElement={<ErrorPage />} />
              <Route path="/" element={<View />} errorElement={<ErrorPage />} />
            </Routes>
          </VariableContextProvider>
        </Content>
      </Layout>
    </>
  );
};

export default observer(Dashboard);
