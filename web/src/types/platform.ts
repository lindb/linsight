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
import { DatasourceSetting } from './datasource';
import { Preference, User } from './user';
import { Chart } from 'chart.js';

import DevIcon from '~devicon/devicon.json';
import { replace } from 'lodash-es';
import { NotificationPlugin } from './alerting';

export interface Bootdata {
  home?: string; // home page
  user: User;
  navTree: Component[];
  datasources: DatasourceSetting[];
  integrations: Integration[];
  preference?: Preference;
  notifications?: NotificationPlugin[];
}

export interface Integration {
  uid: string;
  title: string;
  desc: string;
  icon: string;
  docUrl: string;
}

export interface Component {
  uid: string;
  label: string;
  path: string;
  icon: string;
  role: string;
  component: string;
  children: Component[];
}

export interface OrgComponent {
  componentUid: string;
  role: string;
}

export enum MouseEventType {
  Move = 'move',
  Out = 'out',
  Click = 'click',
}

export interface MouseEvent {
  type: MouseEventType;
  native: any;
  index?: any;
  chart?: Chart;
  series?: any;
  timestamp?: number;
}

export interface IconItem {
  name: string;
}

class IconRepository {
  private icons: Map<string, IconItem> = new Map<string, IconItem>();
  constructor() {
    (DevIcon || []).forEach((i: any) => {
      this.icons.set(i.name, { name: i.name });
      // NOTE: (hack) fix apachekafka->kafka
      this.icons.set(replace(i.name, 'apache', ''), { name: i.name });
    });
  }

  public getIcon(name: string): IconItem | undefined {
    return this.icons.get(name);
  }

  public setIntegrations(integrations: Integration[]) {
    (integrations || []).forEach((i: Integration) => {
      const icon = this.icons.get(i.icon);
      if (icon) {
        this.icons.set(i.uid, icon);
      }
    });
  }

  public getIconCls(name: string, colored: boolean = true): string {
    const item = this.icons.get(name);
    if (!item) {
      return '';
    }
    return `devicon-${item.name}-plain${colored ? ' colored' : ''}`;
  }
}

export const IconRepositoryInst = new IconRepository();
