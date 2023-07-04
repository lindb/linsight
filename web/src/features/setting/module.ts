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
import { Feature, FeatureRepositoryInst } from '@src/types';
import { Datasource, Component, Org, Team, User } from '.';

FeatureRepositoryInst.register(
  new Feature('/setting/datasources/*', 'Datasource Setting', 'Datasource setting', Datasource)
)
  .register(new Feature('/setting/users/*', 'User Setting', 'User setting', User))
  .register(new Feature('/setting/orgs/teams/*', 'Team Setting', 'Team setting', Team))
  .register(new Feature('/setting/orgs/*', 'Org Setting', 'Organization setting', Org))
  .register(new Feature('/setting/components/*', 'Component Setting', 'Component setting', Component))
  .register(new Feature('/setting/orgs/components/*', 'Org. Component Setting', 'Org. component setting', Component));
