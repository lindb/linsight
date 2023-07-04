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
import React, { MutableRefObject, useContext, useMemo, useRef } from 'react';
import { PlatformContext } from '@src/contexts';
import { get } from 'lodash-es';
import TeamList from './team/TeamList';
import { Icon } from '@src/components';
import { useLocation, useNavigate } from 'react-router-dom';
import ListDataSource from './datasource/ListDataSource';
import OrgList from './org/OrgList';
import UserList from './user/UserList';
import ComponentSetting from './component/ComponentSetting';
import OrgComponent from './component/OrgComponent';
import { Component } from '@src/types';
const { Meta } = Card;
const { Text, Title } = Typography;

const Settings = [
  {
    path: '/setting/datasources',
    cmp: <ListDataSource />,
  },
  {
    path: '/setting/users',
    cmp: <UserList />,
  },
  {
    path: '/setting/orgs/teams',
    cmp: <TeamList />,
  },
  {
    path: '/setting/orgs',
    cmp: <OrgList />,
  },
  {
    path: '/setting/components',
    cmp: <ComponentSetting />,
  },
  {
    path: '/setting/orgs/components',
    cmp: <OrgComponent />,
  },
];

const Setting: React.FC = () => {
  const { boot } = useContext(PlatformContext);
  const location = useLocation();
  const navigate = useNavigate();
  const nav = useRef() as MutableRefObject<Map<string, Component>>;

  useMemo(() => {
    nav.current = new Map();
    const buildTree = (navTree: Component[]) => {
      (navTree || []).forEach((item: Component) => {
        nav.current.set(item.path, item);
        if (item.children) {
          buildTree(item.children);
        }
      });
    };
    buildTree(boot.navTree);
  }, [boot]);

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
        {Settings.map((setting: any) => {
          const navItem = nav.current.get(setting.path);
          if (!navItem) {
            return null;
          }
          return (
            <TabPane
              key={setting.path}
              itemKey={setting.path}
              tab={navItem.label}
              icon={<Icon icon={navItem.icon} style={{ marginRight: 8 }} />}>
              {setting.cmp}
            </TabPane>
          );
        })}
      </Tabs>
    </Card>
  );
};
export default Setting;
