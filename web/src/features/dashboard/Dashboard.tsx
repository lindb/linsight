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
import React, { useEffect, useState } from 'react';
import { DashboardSrv } from '@src/services';
import { createSearchParams, Route, Routes, useNavigate, useSearchParams } from 'react-router-dom';
import * as _ from 'lodash-es';
import { Spin, Layout, Typography, Button } from '@douyinfe/semi-ui';
import { IconGridStroked, IconSaveStroked, IconStar, IconSettingStroked, IconStarStroked } from '@douyinfe/semi-icons';
import { DashboardStore, PanelStore } from '@src/stores';
import { Icon, Notification, TimePicker } from '@src/components';
import Setting from './Setting';
import PanelEditor from './PanelEditor';
import View from './View';
import { VisualizationAddPanelType } from '@src/constants';
import { ApiKit } from '@src/utils';
import { observer } from 'mobx-react-lite';
import ViewPanel from './ViewPanel';
import { toJS } from 'mobx';
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

const Dashboard: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const dashboardId = searchParams.get('d');
  const [loading, setLoading] = useState(true);
  const { result: dashboard, loading: isLoading } = useRequest(
    ['load-dashboard', dashboardId],
    async () => {
      if (_.isEmpty(dashboardId)) {
        setLoading(true);
        const id = `abc${new Date().getTime()}`;
        return {
          uid: '',
          title: 'New Dashboard',
          config: {
            panels: [
              {
                title: 'Add panel',
                type: 'addPanel',
                grid: { w: 12, h: 7, x: 0, y: 0, i: id },
                id: id,
              },
            ],
          },
        };
      }
      return DashboardSrv.getDashboard(`${dashboardId}`);
    },
    {}
  );

  useEffect(() => {
    if (!isLoading) {
      console.log('init dashboard...', dashboard);
      DashboardStore.setDashboard(dashboard as any);
      setLoading(false);
    }
  }, [dashboard, isLoading]);

  if (loading) {
    console.log('loading.....');
    return (
      <div style={{ width: '100%', textAlign: 'center', marginTop: 300 }}>
        <Spin size="large" />
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
        <div style={{ marginRight: 12, gap: 8, display: 'flex' }}>
          <Button
            type="tertiary"
            icon={<Icon icon="icon-back2" />}
            onClick={() => {
              console.log('xxxx ddd', toJS(dashboard));
              navigate({
                pathname: '/dashboard',
                search: dashboard.uid ? `${createSearchParams({ d: dashboard.uid })}` : '',
              });
            }}
          />
          <Button
            type="tertiary"
            icon={<Icon icon="icon-panel-add" />}
            onClick={() => {
              DashboardStore.addPanel({
                title: 'Add panel',
                type: VisualizationAddPanelType,
                grid: { w: 12, h: 7, x: 0, y: 0 },
                id: `abc${new Date().getTime()}`,
              });
            }}
          />
          <Button
            type="tertiary"
            icon={<IconSaveStroked />}
            onClick={async () => {
              if (PanelStore.panel) {
                // if has edit panel, need update dashboard's panel
                DashboardStore.updatePanel(PanelStore.panel);
              }
              const success = await DashboardStore.saveDashboard();
              if (success && !dashboardId) {
                // if create dashboard successfully, need set uid to url params
                searchParams.set('d', `${DashboardStore.dashboard.uid}`);
                setSearchParams(searchParams);
              }
            }}
          />
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

//TODO: need add dashboard header?
export default observer(Dashboard);
