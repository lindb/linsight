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
import { DatasourceCategory, DatasourcePlugin, DatasourceRepositoryInst } from '@src/types';
import { LinDBDatasource } from './Datasource';
import LightLogo from './images/logo_light.svg';
import DarkLogo from './images/logo_dark.svg';
import QueryEditor from './QueryEditor';
import { SettingEditor } from './SettingEditor';
import VariableEditor from './VariableEditor';

const LinDB = new DatasourcePlugin(
  DatasourceCategory.Metric,
  'LinDB',
  'lindb',
  'An open-source, distributed time-series database',
  LinDBDatasource
);

LinDB.setSettingEditor(SettingEditor)
  .setQueryEditor(QueryEditor)
  .setVariableEditor(VariableEditor)
  .setDarkLogo(DarkLogo)
  .setLightLogo(LightLogo);
DatasourceRepositoryInst.register(LinDB);
