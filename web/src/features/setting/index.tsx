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
import React from 'react';
import { ErrorPage } from '@src/components';
import { Route, Routes } from 'react-router-dom';
import Setting from './Setting';
import NewTeam from './team/NewTeam';
import './setting.scss';
import EditTeam from './team/EditTeam';
import EditDataSource from './datasource/EditDataSource';

const SettingHome: React.FC = () => {
  return (
    <Routes>
      <Route path="/org/teams/edit/*" element={<EditTeam />} errorElement={<ErrorPage />} />
      <Route path="/org/teams/new" element={<NewTeam />} errorElement={<ErrorPage />} />
      <Route path="/datasource/new" element={<EditDataSource />} errorElement={<ErrorPage />} />
      <Route path="/datasource/edit" element={<EditDataSource />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};
export default SettingHome;
