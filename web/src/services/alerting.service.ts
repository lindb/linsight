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
import {
  NotificationPlugin,
  NotificationSetting,
  ReceiverConfig,
  Notification,
  SearchNotification,
  SearchNotificationResult,
  SearchAlertRuleResult,
  SearchAlertRule,
  AlertRule,
} from '@src/types';
import { ApiKit } from '@src/utils';

const getAllNotificationSettings = (): Promise<NotificationPlugin> => {
  return ApiKit.GET(`${ApiPath.NotificationSetting}`);
};

const saveNotificationSetting = (setting: NotificationSetting): Promise<string> => {
  return ApiKit.PUT(`${ApiPath.NotificationSetting}`, setting);
};

const deleteNotificationSetting = (type: string): Promise<string> => {
  return ApiKit.DELETE(`${ApiPath.NotificationSetting}/${type}`);
};

const testNotify = (receiver: ReceiverConfig): Promise<string> => {
  return ApiKit.PUT(`${ApiPath.Notifications}/test`, receiver);
};

const searchNotifications = (req: SearchNotification): Promise<SearchNotificationResult> => {
  return ApiKit.GET<SearchNotificationResult>(ApiPath.Notifications, req);
};

const createNotification = (notification: Notification): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.Notifications, notification);
};

const updateNotification = (notification: Notification): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.Notifications, notification);
};

const deleteNotification = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.Notifications}/${uid}`);
};

const getNotification = (uid: string): Promise<Notification> => {
  return ApiKit.GET<Notification>(`${ApiPath.Notifications}/${uid}`);
};

const searchAlertRules = (req: SearchAlertRule): Promise<SearchAlertRuleResult> => {
  return ApiKit.GET<SearchAlertRuleResult>(ApiPath.AlertRule, req);
};

const createAlertRule = (rule: AlertRule): Promise<string> => {
  return ApiKit.POST<string>(ApiPath.AlertRule, rule);
};

const updateAlertRule = (rule: AlertRule): Promise<string> => {
  return ApiKit.PUT<string>(ApiPath.AlertRule, rule);
};

const deleteAlertRule = (uid: string): Promise<string> => {
  return ApiKit.DELETE<string>(`${ApiPath.AlertRule}/${uid}`);
};

const getAlertRule = (uid: string): Promise<AlertRule> => {
  return ApiKit.GET<AlertRule>(`${ApiPath.AlertRule}/${uid}`);
};

export default {
  getAllNotificationSettings,
  saveNotificationSetting,
  deleteNotificationSetting,
  testNotify,
  searchNotifications,
  createNotification,
  updateNotification,
  deleteNotification,
  getNotification,
  searchAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  getAlertRule,
};
