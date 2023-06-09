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
import { ThemeType } from '@src/types';

export interface User {
  uid?: string;
  userName: string;
  name: string;
  email: string;
  isDisabled: boolean;
  preference?: Preference;
}

export interface LoginUser {
  username: string;
  password: string;
}

export interface Preference {
  theme?: ThemeType;
  collapsed?: boolean;
  homePage?: string;
}

export interface ChangePassword {
  oldPassword: string;
  newPassword: string;
  confirmPassword: string;
}

export interface ResetPassword {
  userUid: string;
  password: string;
}

export interface UserResult {
  users?: User[];
  total?: number;
}

export interface SearchUser {
  query?: string;
}

export interface UserOrg {
  userUid: string;
  orgUid: string;
  orgName: string;
  role: string;
}

export const RoleList = [
  { label: 'Admin', value: 'Admin', showTick: false },
  { label: 'Editor', value: 'Editor', showTick: false },
  { label: 'Viewer', value: 'Viewer', showTick: false },
];
