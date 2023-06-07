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
  LegendDisplayMode,
  LegendPlacement,
  PanelSetting,
  VisualizationPlugin,
  VisualizationRepositoryInst,
} from '@src/types';
import { Pie } from '@src/plugins/visualizations/pie/Pie';
import { OptionsEditor } from '@src/plugins/visualizations/pie/OptionsEditor';
import Logo from '@src/plugins/visualizations/pie/images/logo2.svg';
import { PieOptions, PieType } from './types';

const pie = new VisualizationPlugin(DatasourceCategory.Metric, 'Pie', 'pie', 'Standard pie chart visualization', Pie);
pie
  .setOptionsEditor(OptionsEditor)
  .datesetTypeFn((_options: PanelSetting): DataSetType => {
    return DataSetType.SingleStat;
  })
  .setDefaultOptions({
    options: {
      pieType: PieType.pie,
      legend: {
        showLegend: true,
        displayMode: LegendDisplayMode.Table,
        placement: LegendPlacement.Right,
        values: ['value', 'percent'],
      },
    } as PieOptions,
  })
  .setDarkLogo(Logo)
  .setLightLogo(Logo);
VisualizationRepositoryInst.register(pie);
