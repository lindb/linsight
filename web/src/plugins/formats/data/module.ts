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
import { FormatCate, FormatRepositoryInst } from '@src/types';
import { BIN, Binary, SI } from './data';

// Data category
FormatRepositoryInst.register(new Binary(FormatCate.Data, 'bytes(IEC)', 'bytes', BIN, 'B', 1024))
  .register(new Binary(FormatCate.Data, 'bytes(SI)', 'decbytes', SI, 'B', 1000))
  .register(new Binary(FormatCate.Data, 'bits(IEC)', 'bits', BIN, 'b', 1024))
  .register(new Binary(FormatCate.Data, 'bits(SI)', 'decbits', SI, 'b', 1000))
  .register(new Binary(FormatCate.Data, 'kibibytes(IEC)', 'kbytes', BIN, 'B', 1024, 1))
  .register(new Binary(FormatCate.Data, 'kibibytes(SI)', 'deckbytes', SI, 'B', 1000, 1))
  .register(new Binary(FormatCate.Data, 'mebibytes(IEC)', 'mbytes', BIN, 'B', 1024, 2))
  .register(new Binary(FormatCate.Data, 'mebibytes(SI)', 'decmbytes', SI, 'B', 1000, 2))
  .register(new Binary(FormatCate.Data, 'gibibytes(IEC)', 'gbytes', BIN, 'B', 1024, 3))
  .register(new Binary(FormatCate.Data, 'gibibytes(SI)', 'decgbytes', SI, 'B', 1000, 3))
  .register(new Binary(FormatCate.Data, 'tebibytes(IEC)', 'tbytes', BIN, 'B', 1024, 4))
  .register(new Binary(FormatCate.Data, 'tebibytes(SI)', 'dectbytes', SI, 'B', 1000, 4))
  .register(new Binary(FormatCate.Data, 'pebibytes(IEC)', 'pbytes', BIN, 'B', 1024, 5))
  .register(new Binary(FormatCate.Data, 'pebibytes(SI)', 'decpbytes', SI, 'B', 1000, 5));

// Data rate category
FormatRepositoryInst.register(new Binary(FormatCate.DataRate, 'packets/sec', 'pps', SI, 'p/s', 1000))
  .register(new Binary(FormatCate.DataRate, 'bytes/sec(IEC)', 'binBps', BIN, 'B/s', 1024))
  .register(new Binary(FormatCate.DataRate, 'bytes/sec(SI)', 'Bps', SI, 'B/s', 1000))
  .register(new Binary(FormatCate.DataRate, 'bits/sec(IEC)', 'binbps', BIN, 'b/s', 1024))
  .register(new Binary(FormatCate.DataRate, 'bits/sec(SI)', 'bps', SI, 'b/s', 1000))
  .register(new Binary(FormatCate.DataRate, 'kibibytes/sec(IEC)', 'KiBs', BIN, 'B/s', 1024, 1))
  .register(new Binary(FormatCate.DataRate, 'kibibytes/sec(SI)', 'KBs', SI, 'B/s', 1000, 1))
  .register(new Binary(FormatCate.DataRate, 'mebibytes/sec(IEC)', 'MiBs', BIN, 'B/s', 1024, 2))
  .register(new Binary(FormatCate.DataRate, 'mebibytes/sec(SI)', 'MBs', SI, 'B/s', 1000, 2))
  .register(new Binary(FormatCate.DataRate, 'gibibytes/sec(IEC)', 'GiBs', BIN, 'B/s', 1024, 3))
  .register(new Binary(FormatCate.DataRate, 'gibibytes/sec(SI)', 'GBs', SI, 'B/s', 1000, 3))
  .register(new Binary(FormatCate.DataRate, 'tebibytes/sec(IEC)', 'TiBs', BIN, 'B/s', 1024, 4))
  .register(new Binary(FormatCate.DataRate, 'tebibytes/sec(SI)', 'TBs', SI, 'B/s', 1000, 4))
  .register(new Binary(FormatCate.DataRate, 'pebibytes/sec(IEC)', 'PiBs', BIN, 'B/s', 1024, 5))
  .register(new Binary(FormatCate.DataRate, 'pebibytes/sec(SI)', 'PBs', SI, 'B/s', 1000, 5));
