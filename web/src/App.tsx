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
import React, { useContext } from 'react';
import { Layout } from '@douyinfe/semi-ui';
import { FeatureMenu, Footer } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { Feature, FeatureRepositoryInst } from '@src/types';
import { Navigate, Route, Routes } from 'react-router-dom';

const Content: React.FC = React.memo(() => {
  const features = FeatureRepositoryInst.getFeatures();
  const { boot } = useContext(PlatformContext);
  return (
    <Routes>
      {features.map((feature: Feature) => {
        const Component = feature.Component;
        return <Route key={feature.Route} path={feature.Route} element={<Component />} />;
      })}
      <Route path="*" element={<Navigate to={boot.home || '/explore'} />} />
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
