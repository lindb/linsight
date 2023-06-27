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
import React, { useCallback, useContext, useMemo, useRef } from 'react';
import { Layout } from '@douyinfe/semi-ui';
import { ErrorPage, FeatureMenu, Footer } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { FeatureRepositoryInst, NavItem } from '@src/types';
import { Navigate, Route, Routes } from 'react-router-dom';
import Explore from './features/explore/Explore';
import User from './features/user/User';

const Content: React.FC = React.memo(() => {
  const { boot } = useContext(PlatformContext);
  const routes = useRef<any[]>([]);
  const addRoutes = useCallback((items: NavItem[]) => {
    items.forEach((item: NavItem) => {
      if (item.path && item.component) {
        const feature = FeatureRepositoryInst.getFeature(item.component);
        if (!feature) {
          return;
        }
        const Component = feature.component;
        routes.current.push(
          <Route key={item.label} path={feature.key} element={<Component />} errorElement={<ErrorPage />} />
        );
      }
      if (item.children) {
        addRoutes(item.children);
      }
    });
  }, []);

  useMemo(() => {
    addRoutes(boot.navTree || []);
  }, [boot, addRoutes]);

  return (
    <Routes>
      {[...routes.current]}
      <Route path="/user/*" element={<User />} errorElement={<ErrorPage />} />
      {/* put /explore route to fix if routes is empty infinite loop*/}
      <Route path="/explore" element={<Explore />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Navigate to={boot.home || '/explore'} />} errorElement={<ErrorPage />} />
    </Routes>
  );
});
Content.displayName = 'Content';

const App: React.FC = () => {
  return (
    <Layout className="linsight">
      <FeatureMenu />
      <Layout>
        <Layout.Content>
          <Content />
        </Layout.Content>
        <Footer />
      </Layout>
    </Layout>
  );
};

export default App;
