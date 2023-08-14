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
import { Span, Trace } from '@src/types';
import { forIn, map, groupBy } from 'lodash-es';
import type { Profile } from '@pyroscope/models/src';

const groupSpans = (span: Span, tTotal: number, tStart: number) => {
  (span.children || []).forEach((x) => groupSpans(x, tTotal, tStart));

  let childrenDur = 0;
  const groups = groupBy(span.children || [], (x) => x.name);
  span.children = map(groups, (group) => {
    const res = group[0];
    for (let i = 1; i < group.length; i += 1) {
      res.duration += group[i].duration;
    }
    childrenDur += res.duration;
    return res;
  });
  span.traceTotal = tTotal;
  span.traceStart = tStart;
  span.total = span.duration || childrenDur;
  span.self = Math.max(0, span.total - childrenDur);
};

const buildTraceTree = (traces: Trace[]) => {
  const spanMap: Map<string, Span> = new Map();
  const tree: Span[] = [];

  (traces || []).forEach((trace: Trace) => {
    const process = trace.process;
    (trace.spans || []).forEach((span: Span) => {
      span.process = process;
      spanMap.set(span.spanId, span);
    });
  });
  spanMap.forEach((span: Span) => {
    if (!span.parentSpanId) {
      tree.push(span);
      return;
    }
    const parentSpan = spanMap.get(span.parentSpanId);
    if (parentSpan) {
      parentSpan.children = parentSpan.children || [];
      parentSpan.children.push(span);
    } else {
      tree.push(span);
    }
  });

  tree.forEach((span: Span) => {
    groupSpans(span, span.duration, span.startTime);
  });

  //FIXME: need process no root

  return tree;
};

const toKeyValueList = (tags: object) => {
  const kvs: any[] = [];
  if (tags != null) {
    forIn(tags, (v: string, k: string) => {
      kvs.push({ key: k, value: v });
    });
  }
  return kvs;
};

const convertTraceToProfile = (traces: Trace[]): Profile => {
  const resultFlamebearer = {
    numTicks: 0,
    maxSelf: 0,
    names: [] as string[],
    levels: [] as number[][],
  };

  const tree = buildTraceTree(traces);

  if (tree) {
    // Step 3: traversing the tree
    function processNode(span: Span, level: number, offset: number) {
      resultFlamebearer.numTicks ||= span.total;
      resultFlamebearer.levels[level] ||= [];
      resultFlamebearer.levels[level].push(offset);
      resultFlamebearer.levels[level].push(span.total);
      resultFlamebearer.levels[level].push(span.self);
      resultFlamebearer.names.push(span.process.serviceName + (span.name || 'total'));
      resultFlamebearer.levels[level].push(resultFlamebearer.names.length - 1);

      (span.children || []).forEach((x: Span) => {
        // fix offset
        offset = x.startTime - x.traceStart;
        offset += processNode(x, level + 1, offset);
        // processNode(x, level + 1, offset);
      });
      return span.total;
    }

    tree.forEach((span: Span) => {
      processNode(span, 0, 0);
    });
  }

  return {
    version: 1,
    flamebearer: resultFlamebearer,
    metadata: {
      format: 'single',
      units: 'trace_samples',
      spyName: 'tracing',
      sampleRate: 1000000000,
    },
  };
};

export default {
  buildTraceTree,
  toKeyValueList,
  convertTraceToProfile,
};
