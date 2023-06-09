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
import { SearchOrg, Org, OrgResult, OrgUser } from '@src/types';
import { ApiKit } from '@src/utils';

const createOrg = (org: Org): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Org, org);
};

const updateOrg = (org: Org): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.Org, org);
};

const deleteOrg = (orgUID: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Org}/${orgUID}`);
};

const fetchOrg = (params: SearchOrg): Promise<OrgResult[]> => {
  return ApiKit.GET<OrgResult[]>(ApiPath.Org, params);
};

const getOrgByUID = (orgUID: string): Promise<Org> => {
  return ApiKit.GET<Org>(`${ApiPath.Org}/${orgUID}`);
};

const getOrgsForCurrentUser = (): Promise<Org[]> => {
  return ApiKit.GET<Org[]>(ApiPath.UserOrg);
};

const getUsersForCurrentOrg = (params: { prefix?: string }): Promise<OrgUser[]> => {
  return ApiKit.GET<OrgUser[]>(ApiPath.OrgUsers, params);
};

export default {
  createOrg,
  updateOrg,
  deleteOrg,
  fetchOrg,
  getOrgByUID,
  getOrgsForCurrentUser,
  getUsersForCurrentOrg,
};
