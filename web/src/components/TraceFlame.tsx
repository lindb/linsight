import React, { MutableRefObject, useEffect, useRef } from 'react';
import FlameChart from 'flame-chart-js';
import * as _ from 'lodash-es';

const TraceFlame: React.FC<{ spans: any }> = (props) => {
  const { spans } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const containRef = useRef() as MutableRefObject<HTMLDivElement | null>;

  const buildFlameData = () => {
    const rootStart = 1669255471995014;
    const convertSpan = (span: any) => {
      const start = span.startTime - rootStart;
      _.set(span, 'start', start);
      _.set(span, 'end', start + span.duration);
      _.set(span, 'name', span.operationName);
      _.set(span, 'type', span.operationName);
      if (span.children) {
        span.children.forEach((child: any) => {
          convertSpan(child);
        });
        span.children = _.sortBy(span.children, ['start']);
      }
    };

    spans.forEach((span: any) => {
      convertSpan(span);
    });
  };
  // getWrapperWH = () => {
  //   const style = window.getComputedStyle(wrapper, null);
  //
  //   return [parseInt(style.getPropertyValue('width')), parseInt(style.getPropertyValue('height')) - 3];
  // };
  useEffect(() => {
    if (!canvasRef.current || !containRef.current || _.isEmpty(spans)) {
      return;
    }
    const canvas = canvasRef.current;
    canvas.width = containRef.current.getBoundingClientRect().width;
    // canvas.height = height;

    buildFlameData();
    console.log(spans, '>>>');
    // spans[0].children = [];
    new FlameChart({
      canvas: canvasRef.current, // mandatory
      data: spans,
      // data: [
      //   {
      //     name: spans[0].name,
      //     start: spans[0].start,
      //     duration: spans[0].duration,
      //     children: [],
      //     // ...spans[0],
      //   },
      //   {
      //     name: spans[0].name,
      //     start: spans[0].start + 100,
      //     duration: spans[0].duration,
      //     children: [],
      //     // ...spans[0],
      //   },
      // ],
      // marks: [
      //   {
      //     shortName: 'DCL',
      //     fullName: 'DOMContentLoaded',
      //     timestamp: 500,
      //     color: '#FFFFFF',
      //   },
      // ],
      colors: {
        task: '#FFFFFF',
        'sub-task': '#000000',
      },
      settings: {
        options: {
          tooltip: () => {
            /*...*/
          }, // see section "Custom Tooltip" below
          timeUnits: 'μs',
        },
      },
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [spans]);
  return (
    <div style={{ width: '100%', flex: 1 }} ref={containRef}>
      <canvas ref={canvasRef} height={500} />
    </div>
  );
};

export default TraceFlame;
