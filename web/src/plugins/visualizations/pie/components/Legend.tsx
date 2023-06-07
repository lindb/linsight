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
import React, { useRef } from 'react';
import { isEmpty, get, upperFirst, sumBy, orderBy } from 'lodash-es';
import { FormatRepositoryInst, Legend as LegendType, LegendDisplayMode, LegendPlacement } from '@src/types';
import classNames from 'classnames';
import { Chart } from 'chart.js';
import { format } from './chart.config';

const LegendHeader: React.FC<{ values: string[] }> = (props) => {
  const { values } = props;
  return (
    <div className="legend-table-header">
      <span className="legend-th-content no-pointer">{/*search input*/}</span>
      {values.map((key) => {
        const headerClass = classNames('legend-th-content', {
          order: false,
          desc: false,
        });

        return (
          <span
            key={key}
            className={headerClass}
            // onClick={() => handleSort(key)}
          >
            <span>{upperFirst(key)}</span>
          </span>
        );
      })}
    </div>
  );
};

const LegendItem: React.FC<{
  series: any;
  values: string[];
  total: number;
  chart: Chart | null;
}> = (props) => {
  const { series, total, values, chart } = props;
  const seriesDiv = useRef<HTMLDivElement>(null);
  const { backgroundColor, label, hidden, value } = series;
  const seriesCls = classNames('legend-series', {
    fade: hidden,
  });
  return (
    <div className={seriesCls}>
      <span className="legend-series-key">
        <i className="legend-series-icon" style={{ backgroundColor: backgroundColor }} />
        <span className="legend-series-label">{label}</span>
      </span>
      {values.map((key: string, index: number) => {
        if (key === 'percent') {
          return (
            <span key={key + index} className="legend-series-value">
              {FormatRepositoryInst.formatString('percentunit', value / total, 2)}
            </span>
          );
        }
        return (
          <span key={key + index} className="legend-series-value">
            {format(chart, value)}
          </span>
        );
      })}
    </div>
  );
};

const Legend: React.FC<{ chart: any; datasets: any }> = (props) => {
  const { chart, datasets } = props;
  const legend = get(chart, 'linsight.extend.legend', {}) as LegendType;
  if (!chart || !legend.showLegend) {
    // if chart not exist or hidden legend, return
    return null;
  }
  const asTable = legend.displayMode === LegendDisplayMode.Table;
  const toRight = legend.placement === LegendPlacement.Right;
  const values = orderBy(legend.values || [], ['desc']);
  const legendCls = classNames('pie-legend', {
    'as-table': asTable,
    'to-right': toRight,
  });
  const legendContentCls = classNames('legend-content', {
    'table-content': asTable,
  });
  const total = sumBy(datasets, (o: any) => o.value);
  return (
    <div className={legendCls}>
      <div className={legendContentCls}>
        {asTable && !isEmpty(datasets) && <LegendHeader values={values} />}
        {datasets.map((series: any, index: number) => {
          return <LegendItem chart={chart} total={total} series={series} key={series.label + index} values={values} />;
        })}
      </div>
    </div>
  );
};

export default Legend;
