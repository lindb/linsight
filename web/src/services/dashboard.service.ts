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
import { Dashboard, SearchDashboard, SearchDashboardResult } from '@src/types';
import { ApiKit } from '@src/utils';

const createDashboard = (dashboard: Dashboard): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Dashboard, dashboard);
};

const updateDashboard = (dashboard: Dashboard): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.Dashboard, dashboard);
};

const getDashboard = (uid: string): Promise<Dashboard> => {
  return ApiKit.GET<Dashboard>(`${ApiPath.Dashboard}/${uid}`);
};

const deleteDashboard = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Dashboard}/${uid}`);
};
const starDashboard = (uid: string): Promise<string> => {
  return ApiKit.PUT<string>(`${ApiPath.Dashboard}/${uid}/star`);
};
const unstarDashboard = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Dashboard}/${uid}/star`);
};

const searchDashboards = (req: SearchDashboard): Promise<SearchDashboardResult> => {
  return ApiKit.GET<SearchDashboardResult>(ApiPath.Dashboard, req);
};

function getDashboardList() {
  return [
    { id: 1, favorite: false, type: 'kubernetes', name: 'Kubernetes workload overview', modified: 'Oct 14 00:00' },
    { id: 2, favorite: true, type: 'linux', name: 'Host system overview', modified: 'Oct 14 00:00' },
    { id: 3, favorite: false, type: 'java', name: 'JVM Metric', modified: 'Oct 14 00:00' },
    { id: 4, favorite: false, type: 'redis', name: 'Redis overview', modified: 'Oct 14 00:00' },
    { id: 5, favorite: false, type: 'go', name: 'Go runtime overview', modified: 'Oct 14 00:00' },
    { id: 6, favorite: true, type: 'docker', name: 'Docker overview', modified: 'Oct 14 00:00' },
    { id: 7, favorite: true, name: 'APM overview', modified: 'Oct 14 00:00' },
  ];
}

function getMetricsList() {
  return [
    {
      id: 1,
      favorite: false,
      type: 'kubernetes',
      name: 'Kubernetes: Cluster cores',
      value: 'builtin:cloud.kubernetes.cluster.cores',
    },
    {
      id: 2,
      favorite: false,
      type: 'kubernetes',
      name: 'Kubernetes: Cluster CPU available, %',
      value: 'builtin:cloud.kubernetes.cluster.cpuAvailableStatistics',
    },
    {
      id: 3,
      favorite: false,
      type: 'kubernetes',
      name: 'Kubernetes: Cluster memory',
      value: 'builtin:cloud.kubernetes.cluster.memory',
    },
    { id: 4, favorite: true, type: 'linux', name: 'CPU usage %', value: 'builtin:host.cpu.usage' },
    { id: 5, favorite: false, type: 'linux', name: 'Memory usage %', value: 'builtin:host.memory.usage' },
    { id: 13, favorite: true, name: 'Order count', value: 'cust:order.service.order.count' },
    {
      id: 6,
      favorite: false,
      type: 'java',
      name: 'Garbage collection count',
      value: 'builtin:jvm.memory.pool.collectionCount',
    },
    {
      id: 7,
      favorite: false,
      type: 'java',
      name: 'Garbage collection suspension time',
      value: 'builtin:jvm.memory.gc.suspensionTime',
    },
    { id: 8, favorite: false, type: 'redis', name: 'Cache hits', value: 'builtin:redis.cache.hits' },
    { id: 9, favorite: false, type: 'go', name: 'Go: Total requests', value: 'builtin:tech.go.http.totalRequests' },
    {
      id: 10,
      favorite: true,
      type: 'docker',
      name: 'Containers: CPU logical cores',
      value: 'builtin:containers.cpu.logicalCores',
    },
    { id: 11, favorite: true, name: 'APM Exception', value: 'builtin:apm.service.exception' },
  ];
}

export default {
  createDashboard,
  updateDashboard,
  deleteDashboard,
  getDashboard,
  searchDashboards,
  starDashboard,
  unstarDashboard,
  getMetricsList,
};
