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
import { FormatRepositoryInst, LegendPlacement, PanelSetting, ThemeType } from '@src/types';
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { Chart, registerables } from 'chart.js';
import { get, set } from 'lodash-es';
import { toJS } from 'mobx';
import { getChartConfig } from './chart.config';
import classNames from 'classnames';
import { PieOptions, PieType } from '../types';
import Legend from './Legend';
import './pie.scss';

Chart.register(...registerables);

export const PieChart: React.FC<{ datasets: any; theme: ThemeType; options: PieOptions; panel: PanelSetting }> = (
  props
) => {
  const { theme, options, datasets, panel } = props;
  const canvasRef = useRef() as MutableRefObject<HTMLCanvasElement | null>;
  const chartInstance = useRef() as MutableRefObject<Chart | null>;
  const [selectedSeries, setSelectedSeries] = useState<string[]>([]);

  useEffect(() => {
    if (!canvasRef.current) {
      // if canvas is null, return it.
      return;
    }

    const labels: any[] = [];
    const data: any[] = [];
    const background: any[] = [];
    (datasets || []).forEach((ds: any) => {
      labels.push(ds.label);
      data.push(ds.value);
      background.push(ds.backgroundColor);
    });
    let chartType = get(options, 'pieType', 'pie');
    if (chartType === PieType.donut) {
      chartType = 'doughnut';
    }
    const chartData = {
      labels: labels,
      datasets: [{ data: data, backgroundColor: background }],
    };

    const chartCfg: any = getChartConfig(theme);
    if (!chartInstance.current) {
      chartCfg.data = chartData;
      chartCfg.type = chartType;
      const canvas = canvasRef.current;
      const chart = new Chart(canvas, chartCfg);
      chartInstance.current = chart;
    } else {
      const chart = chartInstance.current;
      chart.data = chartData;
      set(chart, 'config.type', chartType);
      console.error('charnte........', chartData);

      // update chart after dataset or config changed
      chart.update();
    }
    set(chartInstance.current, 'linsight.extend.legend', options.legend);
    set(chartInstance.current, 'linsight.extend.format', function (val: number) {
      const unit = get(panel, 'fieldConfig.defaults.unit', '');
      const decimals = get(panel, 'field.defaults.decimals', 2);
      return FormatRepositoryInst.get(unit).formatString(val, decimals);
    });
    const currentSelectedSet = new Set<string>(selectedSeries);
    datasets.forEach((series: any) => {
      if (!series.hidden) {
        currentSelectedSet.add(series.label);
      }
    });
    setSelectedSeries(Array.from(currentSelectedSet));
    console.error('rerender pie chart....', chartInstance, chartType, toJS(options), chartData);
  }, [options, datasets, theme, panel]);

  const pieChartCls = classNames('pie-container', {
    'legend-to-right': get(options, 'legend.placement', LegendPlacement.Bottom) === LegendPlacement.Right,
  });

  return (
    <div className={pieChartCls}>
      <div className="pie-canvas">
        <canvas ref={canvasRef} />
      </div>
      <Legend chart={chartInstance.current} datasets={datasets} />
    </div>
  );
};
