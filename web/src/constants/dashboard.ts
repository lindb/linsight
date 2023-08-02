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
import { QuickSelectItem } from '@src/types';

export const DefaultVisualizationType = 'timeseries';
export const VisualizationAddPanelType = 'addPanel';
export const RowPanelType = 'row';
export const DateTimeFormat = 'YYYY-MM-DD HH:mm:ss';
export const DefaultColumns = 24;
export const DefaultRowHeight = 30;
export const AutoRefreshList: QuickSelectItem[] = [
  {
    value: '',
    title: 'off',
  },
  { value: '10', title: `10s` },
  { value: '30', title: `30s` },
  { value: '60', title: `1m` },
  { value: '300', title: `5m` },
];

export const DefaultQuickItem = { title: 'Last 1 hour', value: 'now-1h' };
export const DefaultAutoRefreshItem = { title: 'off', value: '' };
export const QuickSelectList: QuickSelectItem[] = [
  { title: 'Last 15 minutes', value: 'now-15m' },
  { title: 'Last 30 minutes', value: 'now-30m' },
  DefaultQuickItem,
  { title: 'Last 3 hours', value: 'now-3h' },
  { title: 'Last 6 hours', value: 'now-6h' },
  { title: 'Last 12 hours', value: 'now-12h' },
  { title: 'Last 1 day', value: 'now-1d' },
  { title: 'Last 2 days', value: 'now-2d' },
  { title: 'Last 3 days', value: 'now-3d' },
  { title: 'Last 7 days', value: 'now-7d' },
  { title: 'Last 15 days', value: 'now-15d' },
  { title: 'Last 30 days', value: 'now-30d' },
];

export const MixedDatasource = '-- Mixed --';
