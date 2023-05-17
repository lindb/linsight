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
import React, { MutableRefObject, useCallback, useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { get, upperCase, filter } from 'lodash-es';
import classNames from 'classnames';
import { Button, Input } from '@douyinfe/semi-ui';
import { IconSearchStroked } from '@douyinfe/semi-icons';
import { Icon } from '@src/components';
import { Chart } from 'chart.js';
import { CSSKit } from '@src/utils';
import { MouseEventType } from '@src/types';
import {
  getTooltipPositionAndSize,
  handleSeriesClick,
  setPosition,
  TOOLTIP_POSITION,
} from '@src/plugins/visualizations/timeseries/utils/chart';
import { useMouseEvent } from '@src/hooks';
import { format } from './chart.config';

enum SortBy {
  Name = 'name',
  Value = 'value',
}

type Sort = {
  by: SortBy;
  asc: boolean;
};

const TooltipToolbar: React.FC<{
  timestamp?: string;
  search: string;
  onSearch: (val: string) => void;
  sort: Sort;
  selectSort: (val: Sort) => void;
}> = (props) => {
  const { timestamp, search, onSearch, sort, selectSort } = props;
  const [searchVisible, setSearchVisible] = useState(false);
  return (
    <div className="tooltip-toolbar">
      <div className="toolbar-header">
        <div className="tooltip-curr-time">{timestamp}</div>
        <div className="tooltip-btn-group">
          <Button
            size="small"
            className="tooltip-toolbar-btn"
            onClick={() =>
              selectSort({
                by: SortBy.Value,
                asc: sort.by === SortBy.Value ? !sort.asc : true,
              })
            }
            icon={
              <Icon
                icon={
                  sort.by === SortBy.Value ? (sort.asc ? 'icon-number-down' : 'icon-number-up') : 'icon-number-sort'
                }
              />
            }
            theme={sort.by === SortBy.Value ? 'solid' : 'light'}
          />
          <Button
            size="small"
            className="tooltip-toolbar-btn"
            onClick={() =>
              selectSort({
                by: SortBy.Name,
                asc: sort.by === SortBy.Name ? !sort.asc : true,
              })
            }
            icon={
              <Icon
                icon={sort.by === SortBy.Name ? (sort.asc ? 'icon-alpha-down' : 'icon-alpha-up') : 'icon-sort-alpha'}
              />
            }
            theme={sort.by === SortBy.Name ? 'solid' : 'light'}
          />
          <Button
            size="small"
            className="tooltip-toolbar-btn"
            icon={<IconSearchStroked />}
            onClick={() => {
              setSearchVisible(!searchVisible);
              onSearch('');
            }}
          />
        </div>
      </div>
      {(searchVisible || search) && (
        <Input
          className="search-input"
          size="small"
          value={search}
          onChange={(v) => onSearch(v)}
          placeholder="Please input series"
        />
      )}
    </div>
  );
};

const TooltipItem: React.FC<{
  series: any;
  index: number;
  chart: Chart | null;
  onClick: (chart: Chart | null, series: any, event: React.MouseEvent) => void;
}> = (props) => {
  const { series, index, chart, onClick } = props;
  const { borderColor, label, hidden } = series;
  const itemCls = classNames('tooltip-content-list-item', {
    selected: !hidden,
  });

  return (
    <div
      className={itemCls}
      onClick={(e) => {
        onClick(chart, series, e);
      }}>
      <span className="tooltip-series-key">
        <i className="tooltip-series-icon" style={{ background: borderColor }} />
        <span className="tooltip-series-label">{label}</span>
      </span>
      <span className="tooltip-series-value">{format(chart, get(series, `data.[${index}]`, 0))}</span>
    </div>
  );
};

const TooltipContent: React.FC<{
  datasets: any;
  index: number;
  chart: Chart | null;
  sort: Sort;
}> = (props) => {
  const { datasets, index, chart, sort } = props;
  const [selected, setSelected] = useState(false);

  const sortDatasets = () => {
    const d = (datasets || []).sort((a: any, b: any) => {
      if (sort.by === SortBy.Name) {
        return sort.asc ? a.label.localeCompare(b.label) : b.label.localeCompare(a.label);
      } else {
        return sort.asc ? a.data[index] - b.data[index] : b.data[index] - a.data[index];
      }
    });
    return d;
  };

  return (
    <div className="tooltip-list">
      {sortDatasets().map((item: any, idx: number) => (
        <TooltipItem
          key={idx}
          chart={chart}
          series={item}
          index={index}
          onClick={(chart: Chart | null, series: any, event: React.MouseEvent) => {
            handleSeriesClick(chart, series, event);
            setSelected(!selected); // just triger tooltip content render
          }}
        />
      ))}
    </div>
  );
};

const Tooltip: React.FC<{ chart: any }> = (props) => {
  const { chart } = props;
  const canvas = chart?.canvas;

  const { mouseEvent } = useMouseEvent();

  // ref
  const container = useRef() as MutableRefObject<HTMLDivElement>;
  const kick = useRef() as MutableRefObject<HTMLDivElement>;
  const timer = useRef<number | null>();
  const boundaryRect = useRef() as MutableRefObject<DOMRect | null>;
  const chartRect = useRef() as MutableRefObject<DOMRect | null>;
  const currentIndex = useRef() as MutableRefObject<number | null>;

  // state
  const [visible, setVisible] = useState<boolean>(false);
  const [tooltipPosition, setTooltipPosition] = useState<any>(null);
  const [datasets, setDatasets] = useState<any[]>();
  const [search, setSearch] = useState<string>('');
  const [sort, setSort] = useState<Sort>({
    by: SortBy.Name,
    asc: true,
  });

  const removeTooltip = () => {
    if (timer.current) {
      return;
    }
    const timeoutId = +setTimeout(() => {
      setVisible(false);
      timer.current = null;
    }, 200);
    timer.current = timeoutId;
  };

  const keepTooltip = () => {
    if (!timer.current) {
      return;
    }
    clearTimeout(timer.current);
    timer.current = null;
  };

  useEffect(() => {
    if (tooltipPosition) {
      CSSKit.setStyle(container.current, {
        display: 'flex',
      });
    }
  }, [tooltipPosition]);

  const initRect = useCallback(() => {
    if (!canvas) {
      return;
    }
    if (!boundaryRect.current) {
      boundaryRect.current = document.body.getBoundingClientRect();
      chartRect.current = canvas.getBoundingClientRect();
    }
  }, [canvas]);

  const clearRect = () => {
    boundaryRect.current = null;
    chartRect.current = null;
  };

  const handleMouseMove = useCallback(
    (e: MouseEvent) => {
      // disable move
      if (e.metaKey || e.altKey || e.shiftKey || e.ctrlKey) {
        return;
      }

      initRect();

      if (!container.current || !chartRect.current || !boundaryRect.current) {
        return;
      }

      // 计算对应位置及尺寸
      const { position, size, direction } = getTooltipPositionAndSize(
        container.current,
        e.offsetX,
        chartRect.current,
        boundaryRect.current
      );

      // 设置位置
      setPosition(kick.current, position.kick);
      setPosition(container.current, position.container);

      // 设置尺寸
      const { height, width } = size;
      if (height) {
        container.current.style.maxHeight = `${height}px`;
      }
      if (width) {
        container.current.style.maxWidth = `${width}px`;
      }

      // 设置方位
      setTooltipPosition({ position: direction });
      keepTooltip();
    },
    [initRect]
  );

  const handleMouseOut = useCallback((_e: MouseEvent) => {
    removeTooltip();
    clearRect();
  }, []);

  useEffect(() => {
    if (search) {
      setDatasets(filter(get(chart, 'data.datasets', []), (o) => upperCase(o.label).indexOf(upperCase(search)) >= 0));
    } else {
      setDatasets(get(chart, 'data.datasets', []));
    }
  }, [search, chart, visible]);

  useEffect(() => {
    const { type, index, native, chart: chartOfMove } = mouseEvent || ({} as any);
    if (!chart || !native) {
      return;
    }
    switch (type) {
      case MouseEventType.Move:
        if (get(chartOfMove, 'id', 0) != get(chart, 'id', 0)) {
          return;
        }
        setVisible(true);
        currentIndex.current = index;
        handleMouseMove(native);
        return;
      case MouseEventType.Out:
        handleMouseOut(native);
        return;
    }
  }, [mouseEvent, chart, handleMouseOut, handleMouseMove]);

  if (!visible) {
    return null;
  }

  const tooltipCls = classNames('time-series-tooltip', {
    'in-chart':
      tooltipPosition?.position === TOOLTIP_POSITION.LEFT || tooltipPosition?.position === TOOLTIP_POSITION.RIGHT,
  });

  const tooltip = (
    <div className={tooltipCls} ref={container} onMouseMove={keepTooltip} onMouseLeave={removeTooltip}>
      <div ref={kick} className="tooltip-top-kick" />
      <div className="tooltip-title">
        <TooltipToolbar
          timestamp={get(chart, `data.timeLabels[${currentIndex.current}]`, null)}
          search={search}
          onSearch={setSearch}
          sort={sort}
          selectSort={setSort}
        />
      </div>
      <div className="tooltip-content-list">
        <TooltipContent datasets={datasets} index={currentIndex.current || 0} chart={chart} sort={sort} />
      </div>
    </div>
  );

  return createPortal(tooltip, document.body);
};

export default React.memo(Tooltip);
