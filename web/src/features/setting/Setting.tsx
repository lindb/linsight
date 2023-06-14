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
import { Card, Typography, Tag, Tabs, TabPane } from '@douyinfe/semi-ui';
import { IconSettingStroked } from '@douyinfe/semi-icons';
import React, { useContext } from 'react';
import { PlatformContext } from '@src/contexts';
import { get } from 'lodash-es';
import TeamList from './team/TeamList';
import { Icon } from '@src/components';
import { useLocation, useNavigate } from 'react-router-dom';
import ListDataSource from './datasource/ListDataSource';
import OrgList from './org/OrgList';
import MenuSetting from './menu/MenuSetting';
import UserList from './user/UserList';
const { Meta } = Card;
const { Text, Title } = Typography;

const Setting: React.FC = () => {
  const { boot } = useContext(PlatformContext);
  const location = useLocation();
  const navigate = useNavigate();
  return (
    <Card
      className="setting-page"
      bordered={false}
      bodyStyle={{ padding: '0px 24px 24px 24px' }}
      title={
        <Meta
          className="setting-meta"
          title={<Title heading={3}>Setting</Title>}
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Current organization:</Text>
              <Tag>{get(boot, 'user.org.name', 'N/A')}</Tag>
            </div>
          }
          avatar={<IconSettingStroked />}
        />
      }>
      <Tabs
        activeKey={location.pathname}
        tabPaneMotion={false}
        lazyRender
        onChange={(activeKey: string) => {
          navigate({ pathname: activeKey });
        }}>
        <TabPane
          itemKey="/setting/datasources"
          tab="Datasource"
          icon={<Icon icon="datasource" style={{ marginRight: 8 }} />}>
          <ListDataSource />
        </TabPane>
        <TabPane itemKey="/setting/users" tab="User" icon={<Icon icon="user" style={{ marginRight: 8 }} />}>
          <UserList />
        </TabPane>
        <TabPane itemKey="/setting/org/teams" tab="Team" icon={<Icon icon="team" style={{ marginRight: 8 }} />}>
          <TeamList />
        </TabPane>
        <TabPane itemKey="/setting/orgs" tab="Organization" icon={<Icon icon="org" style={{ marginRight: 8 }} />}>
          <OrgList />
        </TabPane>
        <TabPane itemKey="/setting/menu" tab="Menu" icon={<Icon icon="menu" style={{ marginRight: 8 }} />}>
          <MenuSetting />
        </TabPane>
      </Tabs>
    </Card>
  );
};
export default Setting;
