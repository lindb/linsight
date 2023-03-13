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
import React from 'react';
import { Notification } from '@douyinfe/semi-ui';
import { IconAlertTriangle, IconTickCircle } from '@douyinfe/semi-icons';

const error = (err: string): void => {
  Notification.error({
    content: err,
    duration: 5,
    theme: 'light',
    position: 'top',
    icon: <IconAlertTriangle />,
  });
};

const success = (msg: string): void => {
  Notification.success({
    content: msg,
    duration: 5,
    theme: 'light',
    position: 'top',
    icon: <IconTickCircle />,
  });
};

export default { error, success };
