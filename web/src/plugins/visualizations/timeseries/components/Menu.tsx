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
import { Dropdown } from '@douyinfe/semi-ui';
import { Icon } from '@src/components';
import { DatasourceStore, ExemplarStore, PlatformStore } from '@src/stores';
import { MouseEvent, MouseEventType } from '@src/types';
import { Chart } from 'chart.js';
import { reaction } from 'mobx';
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { get, isEmpty } from 'lodash-es';

const Menu: React.FC<{ chart: Chart | null }> = (props) => {
  const { chart } = props;
  // ref
  const chartRect = useRef() as MutableRefObject<DOMRect | undefined>;
  const left = useRef(0);
  const timer = useRef<number | null>();
  const container = useRef() as MutableRefObject<HTMLDivElement>;
  const event = useRef<MouseEvent | null>();

  const [visible, setVisible] = useState<boolean>(false);

  const removeMenu = () => {
    if (timer.current) {
      return;
    }
    const timeoutId = +setTimeout(() => {
      setVisible(false);
      timer.current = null;
    }, 200);
    timer.current = timeoutId;
  };

  useEffect(() => {
    const disposer = reaction(
      () => PlatformStore.mouseEvent,
      (mouseEvent: MouseEvent | null) => {
        const e = mouseEvent?.native;
        const { type, native, chart: chartOfMove } = mouseEvent || ({} as any);
        if (!chart || !native) {
          return;
        }
        if (get(chartOfMove, 'id', 0) != get(chart, 'id', 0)) {
          return;
        }
        // disable click
        if (e.metaKey || e.altKey || e.shiftKey || e.ctrlKey) {
          return;
        }
        if (type === MouseEventType.Click) {
          chartRect.current = chart?.canvas?.getBoundingClientRect();
          left.current = native.offsetX;
          event.current = mouseEvent;
          setVisible(true);
        } else {
          removeMenu();
        }
      }
    );

    return () => disposer();
  }, [chart]);

  const isSupport = (key: string): boolean => {
    const datasourceUID = get(event.current, 'series.meta.target.datasource.uid', '');
    const datasource = DatasourceStore.getDatasource(datasourceUID);
    if (!datasource) {
      return false;
    }
    return !isEmpty(get(datasource, `setting.config.${key}`));
  };

  const keepMenu = () => {
    if (!timer.current) {
      return;
    }
    clearTimeout(timer.current);
    timer.current = null;
  };

  if (!visible || !chartRect.current) {
    return null;
  }

  return createPortal(
    <div
      style={{
        left: left.current + chartRect.current.left,
        top: event.current?.native.offsetY + chartRect.current.top,
        position: 'absolute',
      }}
      ref={container}
      onMouseMove={keepMenu}
      onMouseLeave={removeMenu}>
      <Dropdown
        visible={visible}
        position="right"
        trigger="custom"
        menu={[
          {
            node: 'item',
            name: 'View related traces',
            icon: <Icon icon="tracing" />,
            disabled: !isSupport('traceDatasources'),
            onClick: () => {
              const series = get(event.current, 'series');
              if (series) {
                ExemplarStore.setSelectDataPoint({
                  timestamp: get(event.current, 'timestamp', 0),
                  point: series,
                });
              } else {
                ExemplarStore.setSelectDataPoint(null);
              }
              setVisible(false);
            },
          },
          {
            node: 'item',
            name: 'View related logs',
            icon: <Icon icon="logging" />,
            disabled: !isSupport('loggingDatasources'),
          },
          {
            node: 'item',
            name: 'View related incidents',
            icon: <Icon icon="alerting" />,

            disabled: !isSupport('alertingDatasources'),
          },
          {
            node: 'item',
            name: 'Find correlated metric',
            icon: <Icon icon="metrics" />,
            disabled: !isSupport('metricDatasources'),
          },
        ]}
      />
    </div>,
    document.body
  );
};

export default React.memo(Menu);
