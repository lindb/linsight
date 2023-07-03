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
import { Component, OrgComponent } from '@src/types';
import { ApiKit } from '@src/utils';

const getComponentTree = (): Promise<Component[]> => {
  return ApiKit.GET<Component[]>(ApiPath.Component);
};

const saveOrgComponents = (orgUID: string, cmps: OrgComponent[]): Promise<string> => {
  return ApiKit.PUT<string>(`${ApiPath.Org}/${orgUID}${ApiPath.Component}`, cmps);
};

const getOrgComponents = (orgUID: string): Promise<OrgComponent[]> => {
  return ApiKit.GET<OrgComponent[]>(`${ApiPath.Org}/${orgUID}${ApiPath.Component}`);
};

const createComponent = (cmp: Component): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Component, cmp);
};

const updateComponent = (cmp: Component): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.Component, cmp);
};

const deleteComponentByUID = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Component}/${uid}`);
};

const sortComponents = (cmps: Component[]): Promise<string> => {
  return ApiKit.PUT<string>(`${ApiPath.Component}/sort`, cmps);
};

export default {
  getComponentTree,
  saveOrgComponents,
  getOrgComponents,
  createComponent,
  updateComponent,
  deleteComponentByUID,
  sortComponents,
};
