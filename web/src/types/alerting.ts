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
import { DataQuery } from './datasource';

export interface NotificationPlugin {
  type: string;
  setting: any[];
}

export interface NotificationSetting {
  type: string;
  config: object;
}

export interface ReceiverConfig {
  type: string;
  config: object;
}

export interface Notification {
  name: string;
  receivers: any[];
}

export interface SearchNotification {
  limit?: number;
  offset?: number;
  name?: string;
}

export interface SearchNotificationResult {
  total: number;
  notifications: Notification[];
}

export interface AlertRule {
  title: string;
  version?: number;
  data: DataQuery;
  conditions: any[];
}

export interface SearchAlertRule {
  limit?: number;
  offset?: number;
  title?: string;
}

export interface SearchAlertRuleResult {
  total: number;
  rules: AlertRule[];
}
