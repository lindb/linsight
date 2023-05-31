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
import { PanelSetting } from '@src/types';
import { get } from 'lodash-es';

export interface TimeSeriesOptions {
  drawStyle: 'line' | 'bars' | 'points';
  lineWidth?: number;
  fillOpacity?: number;
  spanNulls?: boolean;
  pointSize?: number;
  showPoints: 'always' | 'never';
  // stepBefore/stepAfter support grafana setting
  lineInterpolation?: 'linear' | 'smooth' | 'step' | 'stepBefore' | 'stepAfter';
  lineStyle?: {
    fill: 'solid' | 'dash' | 'dots';
  };
  axisGridShow?: boolean;
}

export const getCustomOptions = (panel: PanelSetting): TimeSeriesOptions => {
  return get(panel, 'fieldConfig.defaults.custom', {}) as TimeSeriesOptions;
};
