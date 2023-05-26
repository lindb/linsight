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
import { PanelSetting, ThemeType, Thresholds, ThresholdStep } from '@src/types';
import React, { MutableRefObject, useEffect, useRef } from 'react';
import { Chart, registerables } from 'chart.js';
import { get, cloneDeep, orderBy, isNil } from 'lodash-es';
import { toJS } from 'mobx';
import { DefaultChartConfig, getChartThemeConfig } from './chart.config';
import { GaugeOptions } from '../types';

Chart.register(...registerables);

export const GaugeChart: React.FC<{ datasets: any; theme: ThemeType; options: GaugeOptions; panel: PanelSetting }> = (
  props
) => {
  const { theme, options, datasets, panel } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const chartInstance = useRef() as MutableRefObject<Chart | null>;
  const thresholds: Thresholds = get(panel, 'fieldConfig.defaults.thresholds', {});

  const showThresholdMarkers = (): boolean => {
    return options.showThresholdMarkers || false;
  };

  useEffect(() => {
    if (!canvasRef.current) {
      // if canvas is null, return it.
      return;
    }
    const chartType = get(options, 'type', 'doughnut');
    const chartCfg: any = getChartThemeConfig(theme, cloneDeep(DefaultChartConfig));
    const dd = [];
    if (showThresholdMarkers()) {
      const thresholdSteps = orderBy(thresholds.steps || [], [(obj) => isNil(obj.value), 'value'], ['asc', 'asc']);
      const data: any[] = [];
      const backgroundColor: any[] = [];
      thresholdSteps.forEach((step: ThresholdStep) => {
        data.push(step.value);
        backgroundColor.push(step.color);
      });

      dd.push({ data: data, backgroundColor: backgroundColor });
    }
    dd.push({
      data: [90],
      borderWidth: 1,
      backgroundColor: ['green', 'rgba(46,50,56, .1)'],
      weight: 10,
    });
    const datasets = {
      labels: ['10%', '100%'],
      datasets: dd,
    };
    if (!chartInstance.current) {
      chartCfg.data = datasets || [];
      chartCfg.type = 'doughnut';
      // modifyChartConfigs(chartCfg, config);
      // modifyChartOptions(chartCfg.options, config);
      console.log('create new chart', chartInstance, chartCfg, options);
      const canvas = canvasRef.current;
      const chart = new Chart(canvas, chartCfg);
      chartInstance.current = chart;
    } else {
      const chart = chartInstance.current;
      chart.data = datasets || [];
      // theme changed
      chart.options = getChartThemeConfig(theme, {
        options: get(chart, 'options', {}),
      }).options;

      // modifyChartConfigs(chart, config);
      // modifyChartOptions(chart.config.options, config);
      // update chart after dataset or config changed
      console.log('update chart....', chart, datasets, chartType, toJS(options));
      chart.update();
    }
  }, [options, datasets, theme]);

  return (
    <div className="gauge-container">
      <div className="gauge-canvas">
        <div className="percent">34.55%</div>
        <canvas ref={canvasRef} />
      </div>
    </div>
  );
};
