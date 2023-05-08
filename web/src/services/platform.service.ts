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
import { Bootdata } from '@src/types';
import { ApiKit } from '@src/utils';

const boot = (): Promise<Bootdata> => {
  return ApiKit.GET<Bootdata>(ApiPath.Boot);
};

function getProducts() {
  return [
    {
      name: 'Dashboard',
      path: '/dashboard',
      menu: [
        { itemKey: 'dashboard', text: 'Dashboard' },
        { itemKey: 'explore', text: 'Explore' },
        { itemKey: 'metrics', text: 'Metrics' },
      ],
    },
    {
      name: 'APM',
      menu: [
        { itemKey: 'overview', text: 'Overview' },
        { itemKey: 'exception', text: 'Exception' },
      ],
    },
    { name: 'Infrastructure' },
    { name: 'Kubernetes' },
    { name: 'RUM' },
    { name: 'Alert', path: '/alert/alert-list' },
    { name: 'Setting' },
  ];
}

export default {
  boot,
  getProducts,
};
