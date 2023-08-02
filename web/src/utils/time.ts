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
import { DateTimeFormat } from '@src/constants';
import moment from 'moment';

const toHours = (minutes: number): string => {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;

  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}`;
};

const utcOffset = (timeZone: string): string => {
  const currentTimeInTimezone = moment().tz(timeZone);
  const offset = currentTimeInTimezone.utcOffset();
  if (offset == 0) {
    return 'UTC';
  } else if (offset < 0) {
    return `UTC-${toHours(offset * -1)}`;
  }
  return `UTC+${toHours(offset)}`;
};

const parseQuickTime = (timeStr: string): number => {
  const now = new Date();
  const regex = /^now-(\d+)([smhdwMy])$/;
  const match = timeStr.match(regex);

  if (match) {
    const value = parseInt(match[1], 10);
    const unit = match[2];

    switch (unit) {
      case 's':
        now.setSeconds(now.getSeconds() - value);
        break;
      case 'm':
        now.setMinutes(now.getMinutes() - value);
        break;
      case 'h':
        now.setHours(now.getHours() - value);
        break;
      case 'd':
        now.setDate(now.getDate() - value);
        break;
      case 'w':
        now.setDate(now.getDate() - value * 7);
        break;
      case 'M':
        now.setMonth(now.getMonth() - value);
        break;
      case 'y':
        now.setFullYear(now.getFullYear() - value);
        break;
      default:
        throw new Error('Invalid time unit');
    }
  } else {
    throw new Error('Invalid quick time format');
  }

  return now.getTime();
};

const parseTime = (time: string): number => {
  try {
    return parseQuickTime(time);
  } catch (err) {
    return moment(time, DateTimeFormat).valueOf();
  }
};

export default {
  parseQuickTime,
  parseTime,
  toHours,
  utcOffset,
};
