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
import { ThemeType } from '@src/types';
import { ColorKit, CSSKit } from '@src/utils';
import { has, get, set, forIn } from 'lodash-es';
import { TimeSeriesOptions } from '../types';

export const format = (chart: any, val: number) => {
  const formatFn: any = get(chart, 'linsight.extend.format');
  if (formatFn) {
    return formatFn(val);
  }
  return val;
};

const OptionsValueMapping = {
  solid: [],
  dash: [5, 5],
  dots: [1, 5],
  false: false,
  true: true,
  linear: 'default',
  smooth: 'monotone',
};

const ConfigItemMapping = {};

const OptionsMapping = {
  'elements.line.borderWidth': 'lineWidth',
  'elements.line.borderDash': 'lineStyle.fill',
  'elements.line.cubicInterpolationMode': 'lineInterpolation',
  'elements.point.radius': 'pointSize',
  'elements.point.hoverRadius': 'pointSize',
  'scales[x].grid.display': 'axisGridShow',
  'scales[y].grid.display': 'axisGridShow',
  spanGaps: 'spanNulls',
  'scales.y.min': 'min',
  'scales.y.max': 'max',
};

export const modifyChartConfigs = (chart: any, cfg: TimeSeriesOptions) => {
  forIn(ConfigItemMapping, (value: string, key: string) => {
    if (has(cfg, value)) {
      const val = get(cfg, value);
      set(chart, key, get(OptionsValueMapping, val, val));
    }
  });
  forIn(OptionsMapping, (value: string, key: string) => {
    if (has(cfg, value)) {
      const val = get(cfg, value);
      set(chart.options, key, get(OptionsValueMapping, val, val));
    }
  });

  // set fill opacity
  const fillOpacity = cfg.fillOpacity ?? 0;
  if (cfg.drawStyle === 'bars') {
    chart.type = 'bar';
    (chart.data?.datasets || []).forEach((dataset: any) => {
      dataset.pointBackgroundColor = dataset.borderColor;
      dataset.fill = true;
      dataset.backgroundColor = ColorKit.toRGBA(dataset.borderColor, 1);
    });
  } else {
    chart.type = 'line';
    let pointSize = cfg.showPoints === 'always' ? cfg.pointSize : 0;
    if (cfg.drawStyle === 'points') {
      set(chart.options, 'elements.line.borderWidth', 0);
      pointSize = cfg.pointSize ?? 1;
    }
    set(chart.options, 'elements.point.radius', pointSize);
    set(chart.options, 'elements.point.hoverRadius', pointSize);
  }

  if (fillOpacity <= 0) {
    (chart.data?.datasets || []).forEach((dataset: any) => {
      dataset.pointBackgroundColor = dataset.borderColor;
      dataset.fill = false;
    });
  } else {
    (chart.data?.datasets || []).forEach((dataset: any) => {
      dataset.pointBackgroundColor = dataset.borderColor;
      dataset.fill = true;
      dataset.backgroundColor = ColorKit.toRGBA(dataset.borderColor, fillOpacity / 100);
    });
  }
};

export const getChartConfig = (theme: ThemeType) => {
  return {
    type: 'line',
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      zoom: true,
      legend: {
        // custom
      },
      spanGaps: false, // if connect null value
      layout: {
        padding: 0,
      },
      scales: {
        x: {
          type: 'category',
          grid: {
            lineWidth: 0.3,
            tickLength: 0,
            color: CSSKit.getColor('--semi-color-border', theme),
          },
          ticks: {
            color: CSSKit.getColor('--semi-color-text-2', theme),
            font: {
              size: 12,
            },
            maxRotation: 0, // angle in degrees
            callback: function (_value: any, index: number, _values: any) {
              const times = get(this, 'chart.config.data.times', []);
              const labels = get(this, 'chart.config.data.labels', []);
              const interval = get(this, 'chart.config.data.interval', 0);
              if (interval <= 0) {
                return labels[index];
              }
              const time = times[index];
              // TODO: opt?
              if ((time - (time % interval)) % (5 * 60 * 1000) == 0) {
                return labels[index];
              }
              return null;
            },
          },
        },
        y: {
          grid: {
            color: CSSKit.getColor('--semi-color-border', theme),
            tickLength: 0,
          },
          ticks: {
            color: CSSKit.getColor('--semi-color-text-2', theme),
            font: {
              size: 12,
            },
            callback: function (value: any, index: number, _values: any) {
              if (index % 2 == 0) {
                return format(get(this, 'chart'), value);
              }
              return null;
            },
          },
          beginAtZero: true,
        },
      },
      plugins: {
        annotation: {
          annotations: [],
        },
        legend: {
          display: false,
        },
        tooltip: {
          mode: 'dataset',
          enabled: false,
        },
        title: {
          display: false,
        },
      },
      elements: {
        line: {
          tension: false, // disables bezier curve
          borderWidth: 1,
          fillColor: 'rgba(255, 145, 68, 0.2)',
          fill: false,
          cubicInterpolationMode: 'line',
          borderDash: [],
        },
        point: {
          radius: 0,
          borderWidth: 0,
          hoverRadius: 0,
          hoverBorderWidth: 0,
        },
        arc: {
          borderWidth: 0,
        },
      },
      hover: {
        mode: 'index',
        intersect: false,
      },
    },
  };
};
