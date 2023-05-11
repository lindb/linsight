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
import { DatasourceCategory, VisualizationPlugin, VisualizationRepositoryInst } from '@src/types';
import Logo from '@src/plugins/visualizations/timeseries/images/logo.svg';
import { OptionsEditor } from '@src/plugins/visualizations/timeseries/OptionsEditor';
import { TimeSeries } from '@src/plugins/visualizations/timeseries/TimeSeries';
import { LegendMode, Placement } from './types';

const timeseries = new VisualizationPlugin(
  DatasourceCategory.Metric,
  'Time series',
  'timeseries',
  'Time based line, area and bar charts',
  TimeSeries
);
timeseries
  .setOptionsEditor(OptionsEditor)
  .setDefaultOptions({
    options: {
      type: 'line',
      lineWidth: 1,
      lineStyle: 'solid',
      fillOpacity: 0,
      points: 'always',
      pointSize: 1,
      lineInterpolation: 'line',
      spanNulls: 'false',
      legend: {
        mode: LegendMode.List,
        placement: Placement.Bottom,
      },
    },
  })
  .setLightLogo(Logo)
  .setDarkLogo(Logo);
VisualizationRepositoryInst.register(timeseries);
