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
import { ApiPath } from '@src/constants';
import { ChangePassword, LoginUser, Preference, SearchUser, User, UserResult } from '@src/types';
import { ApiKit } from '@src/utils';

const login = (user: LoginUser): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Login, user);
};

const logout = (): Promise<string> => {
  return ApiKit.GET<string>(ApiPath.Logout);
};

const saveUser = (user: User): Promise<User> => {
  return ApiKit.PUT<User>(ApiPath.User, user);
};

const savePreference = (preference: Preference): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.Preference, preference);
};

const getPreference = (): Promise<Preference> => {
  return ApiKit.GET<Preference>(ApiPath.Preference);
};

const changePassword = (changePassword: ChangePassword): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.ChangePassword, changePassword);
};

const createUser = (user: User): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.User, user);
};

const fetchUsers = (params: SearchUser): Promise<UserResult[]> => {
  return ApiKit.GET<UserResult[]>(ApiPath.User, params);
};

const getUserByUID = (uid: string): Promise<User> => {
  return ApiKit.GET<User>(`${ApiPath.User}/${uid}`);
};

const updateUser = (user: User): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.User, user);
};

const disableUserByUID = (uid: string): Promise<string> => {
  return ApiKit.PUT<string>(`${ApiPath.User}/${uid}/disable`);
};

const enableUserByUID = (uid: string): Promise<string> => {
  return ApiKit.PUT<string>(`${ApiPath.User}/${uid}/enable`);
};

export default {
  login,
  logout,
  saveUser,
  getUserByUID,
  createUser,
  fetchUsers,
  updateUser,
  disableUserByUID,
  enableUserByUID,
  getPreference,
  savePreference,
  changePassword,
};
