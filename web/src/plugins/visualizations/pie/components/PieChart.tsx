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
import { LegendPlacement, ThemeType } from '@src/types';
import React, { MutableRefObject, useEffect, useRef } from 'react';
import { Chart, registerables } from 'chart.js';
import { get, set, cloneDeep } from 'lodash-es';
import { toJS } from 'mobx';
import { DefaultChartConfig, getChartThemeConfig, modifyChartConfigs, modifyChartOptions } from './chart.config';
import classNames from 'classnames';

Chart.register(...registerables);

export const PieChart: React.FC<{ datasets: any; theme: ThemeType; config: any }> = (props) => {
  const { theme, config, datasets } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const chartInstance = useRef() as MutableRefObject<Chart | null>;

  useEffect(() => {
    if (!canvasRef.current) {
      // if canvas is null, return it.
      return;
    }
    const chartType = get(config, 'type', 'pie');
    const chartCfg: any = getChartThemeConfig(theme, cloneDeep(DefaultChartConfig));
    if (!chartInstance.current) {
      chartCfg.data = datasets || [];
      chartCfg.type = chartType;
      modifyChartConfigs(chartCfg, config);
      modifyChartOptions(chartCfg.options, config);
      console.log('create new chart', chartInstance, chartCfg, config);
      const canvas = canvasRef.current;
      const chart = new Chart(canvas, chartCfg);
      chartInstance.current = chart;
      // add mouse event handles, after chart created
      // canvas.addEventListener('mousemove', handleMouseMove);
      // canvas.addEventListener('mouseout', handleMouseOut);
      // canvas.addEventListener('click', handleMouseClick);
      // canvas.addEventListener('mousedown', handleMouseDown);
      // canvas.addEventListener('mouseup', handleMouseUp);
    } else {
      const chart = chartInstance.current;
      chart.data = datasets || [];
      // theme changed
      chart.options = getChartThemeConfig(theme, {
        options: get(chart, 'options', {}),
      }).options;

      modifyChartConfigs(chart, config);
      modifyChartOptions(chart.config.options, config);
      // update chart after dataset or config changed
      console.log('update chart....', chart, chartType, toJS(config));
      chart.update();
    }
  }, [config, datasets, theme]);

  const timeseriesChartCls = classNames('time-series-container', {
    'chart-cursor-pointer': true,
    'legend-to-right': get(config, 'legend.placement', LegendPlacement.Bottom) === LegendPlacement.Right,
  });

  return (
    <div className={timeseriesChartCls}>
      <div className="time-series-canvas">
        <canvas
          ref={canvasRef}
          // onMouseEnter={() => {
          //   window.addEventListener('keydown', handleKeyDown);
          // }}
          // onMouseLeave={() => {
          //   window.removeEventListener('keydown', handleKeyDown);
          // }}
        />
      </div>
    </div>
  );
};
