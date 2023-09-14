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
import { Loading } from '@src/components';
import { VariableContextProvider } from '@src/contexts';
import { useRequest } from '@src/hooks';
import { DashboardStore } from '@src/stores';
import { Dashboard } from '@src/components';
import { toJS } from 'mobx';
import React from 'react';

const DashboardView: React.FC<{ dashboardId: string; initVariableValues?: object }> = (props) => {
  const { dashboardId, initVariableValues } = props;
  const { loading } = useRequest(
    ['load-dashboard-dashboard-view', dashboardId],
    async () => {
      return DashboardStore.loadDashbaord(dashboardId);
    },
    {}
  );
  if (loading) {
    return (
      <div className="loading">
        <Loading />
      </div>
    );
  }
  return (
    <VariableContextProvider variables={toJS(DashboardStore.getVariables())} initValues={initVariableValues}>
      <Dashboard readonly />
    </VariableContextProvider>
  );
};

export default DashboardView;
