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
import { ColorKit, ObjectKit } from '@src/utils';
import { has, get, set, forIn, cloneDeep } from 'lodash-es';

export const format = (chart: any, val: number) => {
  const format: any = get(chart, 'linsight.extend.format');
  if (format) {
    return format(val);
  }
  return val;
};

const OptionsValueMapping = {
  solid: [],
  dash: [5, 5],
  false: false,
  true: true,
};

const ConfigItemMapping = {
  'config.type': 'type',
};

const OptionsMapping = {
  'elements.line.borderWidth': 'lineWidth',
  'elements.line.borderDash': 'lineStyle',
  'elements.line.cubicInterpolationMode': 'lineInterpolation',
  'elements.point.radius': 'pointSize',
  'elements.point.hoverRadius': 'pointSize',
  spanGaps: 'spanNulls',
  legend: 'legend',
  'scales.y.min': 'min',
  'scales.y.max': 'max',
};

export const modifyChartConfigs = (chart: any, cfg: object) => {
  forIn(ConfigItemMapping, (value: string, key: string) => {
    if (has(cfg, value)) {
      const val = get(cfg, value);
      set(chart, key, get(OptionsValueMapping, val, val));
      // const val = get(cfg, value);
      // console.log('chart config', key, val, value, isNull(val), isNaN(val));
      // if (isNull(val) || isNaN(val)) {
      //   // unset(chart, key);
      //   console.log('unset...', key, chart, unset(chart, key));
      // } else {
      //   set(chart, key, get(OptionsValueMapping, val, val));
    }
  });
  // set fill opacity
  const fillOpacity = get(cfg, 'fillOpacity', 0);
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

export const modifyChartOptions = (chart: any, options: object) => {
  forIn(OptionsMapping, (value: string, key: string) => {
    if (has(options, value)) {
      const val = get(options, value);
      set(chart, key, get(OptionsValueMapping, val, val));
    }
    // const val = get(options, value);
    // console.log('chart options', key, val, value, isNull(val), isNaN(val));
    // if (isNull(val) || isNaN(val)) {
    //   if (has(CleanOptionsIfNull, key)) {
    //     // unset(chrrt, key);
    //   }
    // } else {
    //   set(chart, key, get(OptionsValueMapping, val, val));
    // }
  });
};

export const DarkChart = {
  options: {
    scales: {
      x: {
        grid: {
          color: 'rgba(255, 255, 255, 0.1)',
        },
        ticks: {
          color: 'rgb(249, 249, 249)',
        },
      },
      y: {
        grid: {
          color: 'rgba(255, 255, 255, 0.1)',
        },
        ticks: {
          color: 'rgb(249, 249, 249)',
        },
      },
    },
  },
};

export const LightChart = {
  options: {
    scales: {
      x: {
        grid: {
          color: 'rgba(232, 233, 234, 1)',
        },
        ticks: {
          color: 'rgba(28, 31, 35, 0.8)',
        },
      },
      y: {
        grid: {
          color: 'rgba(232, 233, 234, 1)',
        },
        ticks: {
          color: 'rgba(28, 31, 35, 0.8)',
        },
      },
    },
  },
};

export function getChartThemeConfig(theme: ThemeType, raw: any) {
  let chartTheme: any = LightChart;
  if (theme === ThemeType.Dark) {
    chartTheme = DarkChart;
  }
  // NOTE: IMPORTANT: need clone object, because merge return target object.
  return cloneDeep(ObjectKit.merge(raw, chartTheme));
}

export const DefaultChartConfig = {
  type: 'line',
  data: {},
  plugins: {
    message: {},
  },
  options: {
    responsive: true,
    maintainAspectRatio: false,
    animation: false,
    zoom: true,
    legend: {
      asTable: true,
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
        },
        ticks: {
          font: {
            size: 12,
          },
          maxRotation: 0, // angle in degrees
          callback: function (_value: any, index: number, _values: any) {
            const times = get(this, 'chart.config.data.times', []);
            const labels = get(this, 'chart.config.data.labels', []);
            if (times[index] % (5 * 60 * 1000) == 0) {
              return labels[index];
            }
            return null;
          },
        },
      },
      y: {
        grid: {
          tickLength: 0,
        },
        ticks: {
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
