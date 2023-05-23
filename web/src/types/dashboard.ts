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
import { PanelSetting, Query } from '@src/types';

export interface Variable {
  type: VariableType;
  name: string;
  label?: string;
  hide: VariableHideType;
  optionType: OptionType[];
  current?: {
    value?: any;
  };
  query?: Query;
}

export interface Dashboard {
  uid?: string;
  title?: string;
  isStarred?: boolean;
  config?: {
    panels?: PanelSetting[];
    variables?: Variable[];
  };
}

export interface SearchDashboard {
  title?: string;
  ownership?: string;
  tags?: string[];
}

export interface SearchDashboardResult {
  total: number;
  dashboards: Dashboard[];
}

export enum VariableHideType {
  LabelAndValue = 0,
  OnlyValue = 1,
  Hide = 2,
}

export enum VariableType {
  Constant = 'constant',
  Query = 'query',
}

export enum OptionType {
  Multi = 'multi',
  All = 'all',
}

export enum SearchParamKeys {
  From = 'from',
  To = 'to',
  Refresh = 'refresh',
}
