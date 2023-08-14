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

const microsecond = 1000;
const millisecond = microsecond * 1000;
const second = millisecond * 1000;
const minute = second * 60;
const hour = minute * 60;
const day = hour * 24;

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

const formatNanoseconds = (nanoseconds: number, decimals: number | undefined = 2): Formatted => {
  if (nanoseconds >= day) {
    return { value: `${round(nanoseconds / day, decimals)}`, suffix: 'day' };
  } else if (nanoseconds >= hour) {
    return { value: `${round(nanoseconds / hour, decimals)}`, suffix: 'hour' };
  } else if (nanoseconds >= minute) {
    return { value: `${round(nanoseconds / minute, decimals)}`, suffix: 'min' };
  } else if (nanoseconds >= second) {
    return { value: `${round(nanoseconds / second, decimals)}`, suffix: 's' };
  } else if (nanoseconds >= millisecond) {
    return { value: `${round(nanoseconds / millisecond, decimals)}`, suffix: 'ms' };
  } else if (nanoseconds >= microsecond) {
    return { value: `${round(nanoseconds / microsecond, decimals)}`, suffix: 'µs' };
  } else {
    return { value: `${round(nanoseconds, decimals)}`, suffix: 'ns' };
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
  toObject,
  formatUnit,
  formatNanoseconds,
  millisecond,
  microsecond,
  second,
  minute,
  hour,
  day,
};
