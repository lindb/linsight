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
import { Popover, List, Typography } from '@douyinfe/semi-ui';
import { Icon } from '@src/components';
import { PlatformStore } from '@src/stores';
import { MouseEvent, MouseEventType } from '@src/types';
import { Chart } from 'chart.js';
import { reaction } from 'mobx';
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import * as _ from 'lodash-es';
const { Text } = Typography;

const Menu: React.FC<{ chart: Chart | null }> = (props) => {
  const { chart } = props;
  // ref
  const chartRect = useRef() as MutableRefObject<DOMRect | undefined>;
  const left = useRef(0);
  const timer = useRef<number | null>();
  const container = useRef() as MutableRefObject<HTMLDivElement>;

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
        if (_.get(chartOfMove, 'id', 0) != _.get(chart, 'id', 0)) {
          return;
        }
        // disable click
        if (e.metaKey || e.altKey || e.shiftKey || e.ctrlKey) {
          return;
        }
        if (type === MouseEventType.Click) {
          chartRect.current = chart?.canvas?.getBoundingClientRect();
          left.current = native.offsetX;
          setVisible(true);
        } else {
          removeMenu();
        }
      }
    );

    return () => disposer();
  }, [chart]);

  const keepMenu = () => {
    if (!timer.current) {
      return;
    }
    clearTimeout(timer.current);
    timer.current = null;
  };

  if (!visible) {
    return null;
  }

  return createPortal(
    <div
      style={{
        left: left.current + chartRect.current?.left,
        top: chartRect.current.y + chartRect.current?.height / 2,
        position: 'absolute',
      }}
      ref={container}
      onMouseMove={keepMenu}
      onMouseLeave={removeMenu}>
      <Popover
        visible={visible}
        position="right"
        trigger="custom"
        content={
          <List
            size="small"
            dataSource={[
              { name: 'View related traces', icon: 'icon-tracing' },
              { name: 'View related containers', icon: 'icon-docker' },
              { name: 'View related logs', icon: 'icon-logging' },
              { name: 'View related incidents', icon: 'icon-alerting' },
              { name: 'Find correlated metric', icon: 'icon-metrics' },
            ]}
            renderItem={(item) => (
              <List.Item style={{ padding: '4px 12px' }}>
                <Text icon={<Icon icon={item.icon} />}> {item.name}</Text>
              </List.Item>
            )}
          />
        }
      />
    </div>,
    document.body
  );
};

export default React.memo(Menu);
