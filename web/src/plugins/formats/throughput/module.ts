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
import { Throughput } from './throughput';

FormatRepositoryInst.register(new Throughput('counts/sec (cps)', 'cps', 'c/s'))
  .register(new Throughput('ops/sec (ops)', 'ops', 'ops/s'))
  .register(new Throughput('requests/sec (rps)', 'reqps', 'req/s'))
  .register(new Throughput('reads/sec (rps)', 'rps', 'rd/s'))
  .register(new Throughput('writes/sec (wps)', 'wps', 'wr/s'))
  .register(new Throughput('I/O ops/sec (iops)', 'iops', 'io/s'))
  .register(new Throughput('counts/min (cpm)', 'cpm', 'c/m'))
  .register(new Throughput('ops/min (opm)', 'opm', 'ops/m'))
  .register(new Throughput('requests/min (rpm)', 'reqpm', 'req/m'))
  .register(new Throughput('reads/min (rpm)', 'rpm', 'rd/m'))
  .register(new Throughput('writes/min (wps)', 'wpm', 'wr/m'));
