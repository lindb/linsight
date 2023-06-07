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
import { get } from 'lodash-es';

export const format = (chart: any, val: number) => {
  const formatFn: any = get(chart, 'linsight.extend.format');
  if (formatFn) {
    return formatFn(val);
  }
  return val;
};

export const getChartConfig = (_theme: ThemeType) => {
  return {
    type: 'pie',
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      zoom: true,
      legend: {
        asTable: true,
      },
      layout: {
        padding: 0,
      },
      plugins: {
        annotation: {
          annotations: [],
        },
        legend: {
          display: false,
        },
        tooltip: {
          enabled: true,
          cornerRadius: 2,
          caretSize: 0,
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
