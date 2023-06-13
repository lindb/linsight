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
import { Button, Card, Form, Radio, TabPane, Tabs, Typography } from '@douyinfe/semi-ui';
import { IconSaveStroked, IconCandlestickChartStroked } from '@douyinfe/semi-icons';
import { PlatformContext } from '@src/contexts';
import { UserSrv } from '@src/services';
import React, { useContext, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Icon } from '@src/components';
import Profile from './Profile';
import ChangePassword from './ChangePassword';

const { Title } = Typography;

const User: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();
  const { boot } = useContext(PlatformContext);
  const { user } = boot;
  return (
    <Card
      className="setting-page"
      bordered={false}
      title={
        <Card.Meta
          className="setting-meta"
          title={<Title heading={3}>{user.name}</Title>}
          description={user.name}
          avatar={<Icon icon="profile" />}
        />
      }>
      <Tabs
        lazyRender
        tabPaneMotion={false}
        activeKey={location.pathname}
        onChange={(activeKey: string) => {
          navigate({ pathname: activeKey });
        }}>
        <TabPane itemKey="/user/profile" tab="Profile" icon={<IconCandlestickChartStroked />}>
          <Profile />
        </TabPane>
        <TabPane
          itemKey="/user/password"
          tab="Change password"
          icon={<Icon icon="password" style={{ marginRight: 8 }} />}>
          <ChangePassword />
        </TabPane>
      </Tabs>
    </Card>
  );
};

export default User;
