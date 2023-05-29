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
import { Formatted } from '@src/types';
import { round, divide, get } from 'lodash-es';

const toObject = (input: string): any => {
  switch (input) {
    case 'false':
      return false;
    case 'true':
      return true;
    default:
      return input;
  }
};

const formatUnit = (value: number | null, units: string[], k = 1000, decimals = 2): Formatted => {
  if (!value || value === 0) {
    return {
      value: '0',
      suffix: get(units, `[0]`, ''),
    };
  }
  const i = Math.min(Math.floor(Math.log(value) / Math.log(k)), units.length - 1);
  const result = round(divide(value, Math.pow(k, i)), decimals);
  const unit = units[i];
  return {
    value: `${result}`,
    suffix: unit,
  };
};

export default {
  formatUnit,
  toObject,
};
