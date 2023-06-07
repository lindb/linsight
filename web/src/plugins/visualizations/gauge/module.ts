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
import {
  DataSetType,
  DatasourceCategory,
  PanelSetting,
  VisualizationPlugin,
  VisualizationRepositoryInst,
} from '@src/types';
import { Gauge } from '@src/plugins/visualizations/gauge/Gauge';
import { OptionsEditor } from '@src/plugins/visualizations/gauge/OptionsEditor';
import Logo from '@src/plugins/visualizations/gauge/images/logo2.svg';
import { DefaultThresholds } from '@src/constants';
import { GaugeOptions, OrientationType } from './types';

const gauge = new VisualizationPlugin(
  DatasourceCategory.Metric,
  'Gauge',
  'gauge',
  'Standard gauge visualization',
  Gauge
);

gauge
  .setOptionsEditor(OptionsEditor)
  .datesetTypeFn((_options: PanelSetting): DataSetType => {
    return DataSetType.SingleStat;
  })
  .setDefaultOptions({
    fieldConfig: {
      defaults: {
        thresholds: DefaultThresholds,
      },
    },
    options: {
      showThresholdMarkers: true,
      orientation: OrientationType.vertical,
    } as GaugeOptions,
  })
  .setDarkLogo(Logo)
  .setLightLogo(Logo);
VisualizationRepositoryInst.register(gauge);
