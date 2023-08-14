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
import { LinGoDatasource } from './Datasource';
import Logo from './images/logo.svg';
import { SettingEditor } from './SettingEditor';

const LinGo = new DatasourcePlugin(
  DatasourceCategory.Trace,
  'LinGo',
  'lingo',
  'An open-source, lightweight tool for building observability pipelines',
  LinGoDatasource
);

LinGo.setSettingEditor(SettingEditor).setDarkLogo(Logo).setLightLogo(Logo);
DatasourceRepositoryInst.register(LinGo);
