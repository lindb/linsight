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
import EditTeam from './team/EditTeam';
import EditDataSource from './datasource/EditDataSource';
import NewOrg from './org/NewOrg';
import EditOrg from './org/EditOrg';
import NewUser from './user/NewUser';
import EditUser from './user/EditUser';

export const Datasource: React.FC = () => {
  return (
    <Routes>
      <Route path="/new" element={<EditDataSource />} errorElement={<ErrorPage />} />
      <Route path="/edit" element={<EditDataSource />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};

export const User: React.FC = () => {
  return (
    <Routes>
      <Route path="/new" element={<NewUser />} errorElement={<ErrorPage />} />
      <Route path="/edit" element={<EditUser />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};

export const Team: React.FC = () => {
  return (
    <Routes>
      <Route path="/edit/*" element={<EditTeam />} errorElement={<ErrorPage />} />
      <Route path="/new" element={<NewTeam />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};

export const Org: React.FC = () => {
  return (
    <Routes>
      <Route path="/new" element={<NewOrg />} errorElement={<ErrorPage />} />
      <Route path="/edit" element={<EditOrg />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};

export const Component: React.FC = () => {
  return (
    <Routes>
      <Route path="/*" element={<Setting />} errorElement={<ErrorPage />} />
    </Routes>
  );
};
