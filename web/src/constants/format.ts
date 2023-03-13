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
import { FormatKit } from '@src/utils';

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
export const FormatCategories = [
  {
    label: 'Misc',
    value: 'misc',
    key: 'misc',
    children: [
      { label: 'Short', value: 'short', key: 'short' },
      { label: 'Percent (0~100)', value: 'precent', key: 'precent', formatter: FormatKit.transformPercent },
      {
        label: 'Percent (0.0~1.0)',
        value: 'precentunit',
        key: 'precentunit',
        formatter: FormatKit.transformPercentUnit,
      },
    ],
  },
  {
    label: 'Data',
    value: 'data',
    key: 'data',
    children: [
      { label: 'Bytes(IEC)', value: 'bytes', key: 'bytes' },
      { label: 'Bytes(SI)', value: 'descbytes', key: 'descbytes' },
    ],
  },
  {
    label: 'Data Rate',
    value: 'datarate',
    key: 'datarate',
    children: [
      { label: 'Packets/sec', value: 'pps', key: 'pps' },
      { label: 'Bytes/sec(SI)', value: 'binBps', key: 'binBps' },
    ],
  },
  {
    label: 'Time',
    value: 'time',
    key: 'time',
    children: [
      { label: 'Nanoseconds (ns)', value: 'ns', key: 'ns' },
      { label: 'Microseconds (µs)', value: 'µs', key: 'µs' },
    ],
  },
  {
    label: 'Throughtput',
    value: 'throughtput',
    key: 'throughtput',
    children: [
      { label: 'Counts/sec (cps)', value: 'cps', key: 'cps' },
      { label: 'Ops/sec (ops)', value: 'ops', key: 'ops' },
      { label: 'Requests/sec (rps)', value: 'rps', key: 'rps' },
    ],
  },
];
