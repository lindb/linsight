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
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { Legend } from '@src/plugins/visualizations/timeseries/components/Legend';
import Tooltip from '@src/plugins/visualizations/timeseries/components/Tooltip';
import { getChartConfig, modifyChartConfigs } from '@src/plugins/visualizations/timeseries/components/chart.config';
import { Chart, registerables } from 'chart.js';
import { cloneDeep, get, set, find, isEmpty } from 'lodash-es';
import classNames from 'classnames';
import {
  FormatRepositoryInst,
  LegendPlacement,
  MouseEventType,
  PanelSetting,
  SearchParamKeys,
  ThemeType,
} from '@src/types';
import { PlatformStore } from '@src/stores';
import { CSSKit } from '@src/utils';
import annotationPlugin from 'chartjs-plugin-annotation';
import { getCustomOptions } from '../types';
import moment from 'moment';
import { DateTimeFormat } from '@src/constants';
import { useSearchParams } from 'react-router-dom';

Chart.register(annotationPlugin);
Chart.register(...registerables);

const Zoom = {
  drag: false,
  isMouseDown: false,
  selectedStart: 0,
  selectedEnd: 0,
  x: 0,
};

export const TimeSeriesChart: React.FC<{ datasets: any; theme: ThemeType; panel: PanelSetting }> = (props) => {
  const { theme, panel, datasets } = props;
  const [searchParams, setSearchParams] = useSearchParams();

  const [selectedSeries, setSelectedSeries] = useState<string[]>([]);
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const chartInstance = useRef() as MutableRefObject<Chart | null>;
  const zoomRef = useRef(cloneDeep(Zoom));
  const zoomDivRef = useRef() as MutableRefObject<HTMLDivElement>;
  const crosshairRef = useRef() as MutableRefObject<HTMLDivElement>;

  const currPointIndex = useRef(-1);

  const isZoom = (chart: Chart | null) => {
    return get(chart, 'options.zoom', false);
  };

  const resetZoomRange = () => {
    if (isZoom(chartInstance.current)) {
      CSSKit.setStyle(zoomDivRef.current, {
        display: 'none',
        width: '0px',
      });
      zoomRef.current.isMouseDown = false;
      zoomRef.current.drag = false;
      zoomRef.current.selectedStart = 0;
      zoomRef.current.selectedEnd = 0;
      zoomRef.current.x = 0;
    }
  };

  const handleMouseMove = (e: MouseEvent) => {
    if (!chartInstance.current) {
      return;
    }
    const chart = chartInstance.current;
    const points: any = chart.getElementsAtEventForMode(e, 'index', { intersect: false }, false);
    if (!points || points.length <= 0) {
      return;
    }
    const currIdx = points[0].index;
    const interval = get(chart, 'config.data.interval', 0) / 1000;
    const v = currIdx * interval;
    currPointIndex.current = currIdx;

    if (zoomRef.current.isMouseDown && isZoom(chart)) {
      const chartArea = chart.chartArea;
      const datasets = get(chart, 'data', {}) as any;
      zoomRef.current.selectedEnd = datasets.times[points[0].index];
      zoomRef.current.drag = true;
      const width = e.offsetX - zoomRef.current.x;
      if (width >= 0) {
        CSSKit.setStyle(zoomDivRef.current, {
          width: `${width}px`,
        });
      } else {
        CSSKit.setStyle(zoomDivRef.current, {
          width: `${-width}px`,
          transform: `translate(${e.offsetX}px, ${chartArea.top}px)`,
        });
      }
    }

    // move cross hair
    for (let key of Object.keys(Chart.instances)) {
      const currChart = Chart.instances[`${key}`];
      const crosshair = get(currChart, 'linsight.extend.crosshair', null);
      const len = get(currChart, 'config.data.times', []).length;
      if (!crosshair || len == 0) {
        continue;
      }

      const chartArea = currChart.chartArea;
      const width = get(currChart, 'chartArea.width', 0);
      const i = get(currChart, 'config.data.interval', 0) / 1000;

      const x = (v / ((len - 1) * i)) * width + chartArea.left;
      if (x > chartArea.right) {
        continue;
      }
      const top = chartArea.top;
      const bottom = chartArea.bottom;
      CSSKit.setStyle(crosshair, {
        display: 'block',
        height: `${bottom - top}px`,
        transform: `translate(${x}px, ${top}px)`,
      });
    }
    // set platform state context
    PlatformStore.setMouseEvent({
      type: MouseEventType.Move,
      index: currIdx,
      native: e,
      chart: chartInstance.current,
    });
  };

  const handleMouseClick = (e: MouseEvent) => {
    console.log('xxxxxx..... cccccc');
    PlatformStore.setMouseEvent({
      type: MouseEventType.Click,
      native: e,
    });
  };

  const handleMouseOut = (e: MouseEvent) => {
    currPointIndex.current = -1;
    try {
      resetZoomRange();
      for (let key of Object.keys(Chart.instances)) {
        const currChart = Chart.instances[`${key}`];
        const crosshair = get(currChart, 'linsight.extend.crosshair', null);
        if (!crosshair) {
          continue;
        }

        CSSKit.setStyle(crosshair, {
          display: 'none',
        });
      }
    } finally {
      PlatformStore.setMouseEvent({
        type: MouseEventType.Out,
        native: e,
      });
    }
  };

  const handleMouseDown = (e: MouseEvent) => {
    if (!chartInstance.current) {
      return;
    }
    const chart = chartInstance.current;
    const datasets = get(chart, 'data', {}) as any;
    if (!isEmpty(datasets) && isZoom(chart)) {
      zoomRef.current.isMouseDown = true;
      const points: any = chart.getElementsAtEventForMode(e, 'index', { intersect: false }, false);
      if (points && points.length > 0) {
        zoomRef.current.selectedStart = datasets.times[points[0].index];
      }
      zoomRef.current.x = e.offsetX;
      const chartArea = chart.chartArea;
      const height = chartArea.height;
      CSSKit.setStyle(zoomDivRef.current, {
        display: 'block',
        height: `${height}px`,
        transform: `translate(${zoomRef.current.x}px, ${chartArea.top}px)`,
      });
    }
  };

  const handleMouseUp = (_e: MouseEvent) => {
    if (zoomRef.current.drag) {
      const start = Math.min(zoomRef.current.selectedStart, zoomRef.current.selectedEnd);
      const end = Math.max(zoomRef.current.selectedStart, zoomRef.current.selectedEnd);
      const from = moment(start).format(DateTimeFormat);
      const to = moment(end).format(DateTimeFormat);
      searchParams.set(SearchParamKeys.From, from);
      searchParams.set(SearchParamKeys.To, to);
      setSearchParams(searchParams);
    }
    resetZoomRange();
  };

  const handleKeyDown = (e: any) => {
    if (!e.ctrlKey || (e.keyCode !== 76 && e.keyCode !== 85)) {
      return;
    }
    // ctrl+l => lock crosshair
    if (!chartInstance.current || currPointIndex.current < 0) {
      return;
    }
    const currIdx = currPointIndex.current;
    for (let key of Object.keys(Chart.instances)) {
      const currChart = Chart.instances[`${key}`];
      const times = get(currChart, 'config.data.labels', []);
      const len = times.length;

      if (len == 0 || len <= currIdx) {
        //FIXME: diff interval
        continue;
      }
      const annotations: any = get(currChart, 'options.plugins.annotation.annotations');
      const lock: any = find(annotations, (n: any) => {
        return n.id === 'lock-crosshair';
      });
      if (lock) {
        if (e.keyCode === 76) {
          lock.display = true;
          lock.value = currIdx;
        } else {
          lock.display = false;
        }
      } else {
        annotations.push({
          id: 'lock-crosshair',
          type: 'line',
          borderColor: 'rgba(252,114,98,1)',
          borderWidth: 1,
          borderDash: [2, 2],
          label: {
            display: false,
            content: 'L',
            position: 'start',
          },
          scaleID: 'x',
          value: currIdx,
        });
      }
      currChart.update();
    }
  };

  useEffect(() => {
    if (!canvasRef.current) {
      // if canvas is null, return it.
      return;
    }
    const customOptions = getCustomOptions(panel);
    const chartCfg: any = getChartConfig(theme);
    chartCfg.options.legend = get(panel, 'options.legend', {});
    console.error('chart cfg', chartCfg, customOptions);
    if (!chartInstance.current) {
      chartCfg.data = datasets || [];
      modifyChartConfigs(chartCfg, customOptions);
      // console.log('create new chart', chartInstance, chartCfg, options);
      const canvas = canvasRef.current;
      const chart = new Chart(canvas, chartCfg);
      set(chart, 'linsight.extend.crosshair', crosshairRef.current);
      chartInstance.current = chart;
      // add mouse event handles, after chart created
      canvas.addEventListener('mousemove', handleMouseMove);
      canvas.addEventListener('mouseout', handleMouseOut);
      canvas.addEventListener('click', handleMouseClick);
      canvas.addEventListener('mousedown', handleMouseDown);
      canvas.addEventListener('mouseup', handleMouseUp);
    } else {
      const chart = chartInstance.current;
      chart.data = datasets || [];
      // theme changed
      chart.options = chartCfg.options;

      modifyChartConfigs(chart.config, customOptions);
      // update chart after dataset or config changed
      chart.update();
    }

    set(chartInstance.current, 'linsight.extend.format', function (val: number) {
      const unit = get(panel, 'fieldConfig.defaults.unit', '');
      const decimals = get(panel, 'field.defaults.decimals', 2);
      return FormatRepositoryInst.get(unit).formatString(val, decimals);
    });

    const chartDatasets = get(chartInstance.current, 'data.datasets', []);
    const currentSelectedSet = new Set<string>(selectedSeries);
    chartDatasets.forEach((series: any) => {
      if (!series.hidden) {
        currentSelectedSet.add(series.label);
      }
    });
    setSelectedSeries(Array.from(currentSelectedSet));
    console.error('re-render time series chart');
  }, [datasets, theme, panel]);

  /**
   * destroy resource
   */
  useEffect(() => {
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      if (chartInstance.current) {
        const canvas = chartInstance.current.canvas;
        canvas.removeEventListener('mousemove', handleMouseMove);
        canvas.removeEventListener('mouseout', handleMouseOut);
        canvas.removeEventListener('click', handleMouseClick);
        canvas.removeEventListener('mousedown', handleMouseDown);
        canvas.removeEventListener('mouseup', handleMouseUp);
        chartInstance.current.destroy();
        chartInstance.current = null;
      }
    };
  }, []);

  const timeseriesChartCls = classNames('time-series-container', {
    'chart-cursor-pointer': true,
    'legend-to-right': get(panel, 'options.legend.placement') === LegendPlacement.Right,
  });

  return (
    <div className={timeseriesChartCls}>
      <div className="time-series-canvas">
        <canvas ref={canvasRef} />
        <div ref={crosshairRef} className="crosshair" />
        <div ref={zoomDivRef} className="zoom" />
      </div>
      <Legend chart={chartInstance.current} />
      <Tooltip chart={chartInstance.current} />
    </div>
  );
};
