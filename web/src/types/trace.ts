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

export interface Event {
  name: string;
  timestamp: number; // ns
  tags: object;
}

export interface Link {
  traceId: string;
  spanId: string;
  traceState: string;
  tags: object;
}

export interface Process {
  serviceName: string;
  instanceId: string;
  serviceVersion: string;
  sdk: string;
  sdkLanguage: string;
  sdkVersion: string;
  tags: object;
}

export interface Span {
  traceId: string;
  parentSpanId: string;
  spanId: string;
  traceState: string;
  name: string;
  kind: SpanKind;
  startTime: number; // ns
  endTime: number; // ns
  duration: number; // ns
  tags: object;

  events: Event[];
  links: Link[];
  traceTotal: number;
  traceStart: number;
  total: number;
  self: number;

  // client build
  children: Span[];
  process: Process;
}

export interface Trace {
  process: Process;
  spans: Span[];
}

export interface Exemplar {
  traceId: string;
  spanId: string;
  duration: number;
}

export enum SpanKind {
  Unspecified = 'Unspecified',
  Internal = 'Internal',
  Server = 'Server',
  Client = 'Client',
  Producer = 'Producer',
  Consumer = 'Consumer',
}

// ref: https://github.com/open-telemetry/opentelemetry-collector/blob/main/semconv/v1.18.0/generated_trace.go
export enum TraceAttributes {
  DBSystem = 'db.system',
  RPCSystem = 'rpc.system',
  MessagingSystem = 'messaging.system',
  HTTPScheme = 'http.scheme',
}
