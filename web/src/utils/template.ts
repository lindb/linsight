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
import { get, trim } from 'lodash-es';

const template = (tmpl: string, params?: object): string => {
  const matches = tmpl.match(/\$\{\s*(\w+)\s*\}/);
  if (matches) {
    return get(params, trim(matches[1]), '');
  }
  return tmpl;
};

const findTemplateName = (tmpl: string): string => {
  const matches = tmpl.match(/\$\{\s*(\w+)\s*\}/);
  if (matches) {
    return matches[1];
  }
  return '';
};

export default { template, findTemplateName };
