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
import { FormatRepositoryInst, PanelSetting, ThemeType, Threshold } from '@src/types';
import React, { MutableRefObject, useCallback, useEffect, useRef, useState } from 'react';
import { Chart, registerables } from 'chart.js';
import { get, cloneDeep } from 'lodash-es';
import { toJS } from 'mobx';
import { DefaultChartConfig } from './chart.config';
import { GaugeOptions } from '../types';
import { CSSKit, FieldKit } from '@src/utils';

Chart.register(...registerables);

export const GaugeChart: React.FC<{ dataset: any; theme: ThemeType; options: GaugeOptions; panel: PanelSetting }> = (
  props
) => {
  const { theme, options, dataset, panel } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const chartInstance = useRef() as MutableRefObject<Chart | null>;
  const [activeThreshold, setActiveThreshold] = useState<Threshold>({});
  const [fontSize, setFontSize] = useState('1rem');
  const fieldConfig = get(panel, 'fieldConfig.defaults', {});

  const showThresholdMarkers = useCallback((): boolean => {
    return options.showThresholdMarkers || false;
  }, [options]);

  useEffect(() => {
    if (!canvasRef.current) {
      // if canvas is null, return it.
      return;
    }
    const chartCfg: any = cloneDeep(DefaultChartConfig);
    const dd = [];
    const formattedThresholds = FieldKit.getFormattedThresholds(fieldConfig);
    if (showThresholdMarkers()) {
      const data: any[] = [];
      const backgroundColor: any[] = [];
      formattedThresholds.forEach((step: Threshold) => {
        data.push(step._percent);
        backgroundColor.push(step.color);
      });
      dd.push({ data: data, backgroundColor: backgroundColor });
    }
    const value = dataset.value;
    const activeThreshold = FieldKit.getActiveThreshold(value, toJS(formattedThresholds)) || {};

    const diff = formattedThresholds[formattedThresholds.length - 1].value || 0 - value;
    dd.push({
      data: [value, diff > 0 ? diff : 0],
      borderWidth: 1,
      backgroundColor: [activeThreshold.color, CSSKit.getColor('--semi-color-fill-0', theme)],
      weight: 10,
      borderColor: CSSKit.getColor('--semi-color-bg-1', theme),
    });
    const datasets = {
      datasets: dd,
    };
    if (!chartInstance.current) {
      chartCfg.data = datasets;
      chartCfg.type = 'doughnut';
      const canvas = canvasRef.current;
      const chart = new Chart(canvas, chartCfg);
      chartInstance.current = chart;
    } else {
      const chart = chartInstance.current;
      chart.data = datasets;
      // update chart after dataset or config changed
      chart.update();
    }
    console.error('render gauge chart');
    setActiveThreshold(activeThreshold);
    const chartArea = chartInstance.current.chartArea;
    setFontSize(`${Math.min(chartArea.height, chartArea.width) / 10}px`);
  }, [options, dataset, theme, fieldConfig, showThresholdMarkers]);

  return (
    <div className="gauge-container">
      <div className="gauge-canvas">
        <div className="gauge-text" style={{ fontSize: `${fontSize}`, color: activeThreshold.color }}>
          {FormatRepositoryInst.get(get(fieldConfig, 'unit', '')).formatString(dataset.value)}
        </div>
        <canvas ref={canvasRef} />
        <div className="gauge-label">{dataset.label}</div>
      </div>
    </div>
  );
};
