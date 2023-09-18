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
import { ErrorPage } from '@src/components';
import React from 'react';
import { Route, Routes } from 'react-router-dom';
import { NotificationList } from './NotificationList';
import { NotificationSetting } from './NotificationSetting';
import { RuleList } from './RuleList';
import { RuleSetting } from './RuleSetting';

export const AlertRule: React.FC = () => {
  return (
    <Routes>
      <Route path="/new" element={<RuleSetting />} errorElement={<ErrorPage />} />
      <Route path="/edit" element={<RuleSetting />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<RuleList />} errorElement={<ErrorPage />} />
    </Routes>
  );
};

export const Notification: React.FC = () => {
  return (
    <Routes>
      <Route path="/new" element={<NotificationSetting />} errorElement={<ErrorPage />} />
      <Route path="/edit" element={<NotificationSetting />} errorElement={<ErrorPage />} />
      <Route path="/*" element={<NotificationList />} errorElement={<ErrorPage />} />
    </Routes>
  );
};
