import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';
import cytoscape from 'cytoscape';
import * as _ from 'lodash-es';

const TraceMap: React.FC<{ spans: any; processes: any }> = (props) => {
  const { processes, spans } = props;
  console.log(spans, processes);
  const containRef = useRef() as MutableRefObject<HTMLDivElement | null>;

  const buildDAG = useCallback(() => {
    const rootStart = 1669255471995014;
    const rs: any[] = [];
    const convertSpan = (span: any, parent: string) => {
      const start = span.startTime - rootStart;
      const process: any = _.get(processes, span.processID, {});
      const language = _.get(
        _.find(
          _.get(process, 'tags', []),
          (o: any) => o.key === 'telemetry.sdk.language' || o.key === 'library.language'
        ),
        'value',
        ''
      );
      const spanKind = _.find(span.tags, (o: any) => o.key === 'span.kind');
      const serviceName = _.get(process, 'serviceName');
      console.log(language, spanKind, span);
      if (!_.find(rs, { data: { id: `${serviceName}:${span.operationName}` } })) {
        if (serviceName) {
          const p = _.find(rs, { data: { id: parent } });
          if (p) {
            rs.push({ data: { source: parent, target: `${serviceName}:${span.operationName}` } });
          }
        }
        rs.push({
          data: { id: `${serviceName}:${span.operationName}`, label: `${serviceName}:${span.operationName}` },
          style: {
            'background-image': `url(/${language}.png)`,
            'background-size': '60px 60px',
          },
        });
      }
      if (span.children) {
        span.children.forEach((child: any) => {
          convertSpan(child, `${serviceName}:${span.operationName}`);
        });
        span.children = _.sortBy(span.children, ['start']);
      }
    };

    spans.forEach((span: any) => {
      convertSpan(span, '');
    });
    console.log('xxxxx....', rs);
    return rs;
  }, [spans, processes]);

  useEffect(() => {
    if (!containRef.current) {
      return;
    }
    var cy = cytoscape({
      container: containRef.current,
      style: cytoscape
        .stylesheet()
        .selector('edge')
        .css({
          'curve-style': 'bezier',
          width: 3,
          'target-arrow-shape': 'triangle',
          'line-color': 'rgb(58,179,70)',
          'target-arrow-color': 'rgb(58,179,70)',
        })
        .selector('node')
        .css({
          height: 80,
          width: 80,
          'background-fit': 'cover',
          'border-color': 'rgb(58,179,70)',
          'border-width': 3,
          'border-opacity': 0.5,
          'background-color': '#fff',
          opacity: 1,
        })
        .selector('node[label]')
        .style({ label: 'data(label)', 'text-valign': 'bottom', 'padding-bottom': 12 }),
      elements: buildDAG(),
      //
      // elements: [
      //   { data: { id: 'a' } },
      //   {
      //     data: { id: 'bird' },
      //     style: {},
      //   },
      //   {
      //     data: {
      //       id: 'ab',
      //       source: 'a',
      //       target: 'bird',
      //     },
      //   },
      // ],
      layout: {
        // name: 'grid',
        // cols: 3,
      },
    });
    console.log('xxxxxmap');
  }, [buildDAG]);
  return (
    <div>
      <div ref={containRef} style={{ width: '100%', height: 400 }}></div>
    </div>
  );
};

export default TraceMap;
