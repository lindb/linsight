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
  OrientationType,
  PanelSetting,
  VisualizationPlugin,
  VisualizationRepositoryInst,
} from '@src/types';
import { OptionsEditor } from './OptionsEditor';
import Logo from './images/logo.svg';
import Stat from './Stat';
import { ColorMode, JustifyMode, StatOptions, TextMode } from './types';

const stat = new VisualizationPlugin(DatasourceCategory.Metric, 'Stat', 'stat', 'Big stat values', Stat);
stat
  .setOptionsEditor(OptionsEditor)
  .datesetTypeFn((_options: PanelSetting): DataSetType => {
    return DataSetType.SingleStat;
  })
  .setDefaultOptions({
    options: {
      orientation: OrientationType.vertical,
      colorMode: ColorMode.none,
      textMode: TextMode.valueAndName,
      justifyMode: JustifyMode.auto,
    } as StatOptions,
  })
  .setDarkLogo(Logo)
  .setLightLogo(Logo);
VisualizationRepositoryInst.register(stat);
