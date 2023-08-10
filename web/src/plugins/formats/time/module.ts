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
import { FormatRepositoryInst } from '@src/types';
import { FormatKit } from '@src/utils';
import { Time } from './time';

FormatRepositoryInst.register(new Time('nanoseconds (ns)', 'ns', 1))
  .register(new Time('microseconds (µs)', 'µs', FormatKit.microsecond))
  .register(new Time('milliseconds (ms)', 'ms', FormatKit.millisecond))
  .register(new Time('seconds (s)', 's', FormatKit.second))
  .register(new Time('minutes (m)', 'm', FormatKit.minute))
  .register(new Time('hours (h)', 'h', FormatKit.hour))
  .register(new Time('days (d)', 'd', FormatKit.day));
