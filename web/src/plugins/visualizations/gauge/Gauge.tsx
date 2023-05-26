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
import { VisualizationProps } from '@src/types';
import { DataSetKit } from '@src/utils';
import React, { useEffect, useState } from 'react';
import { GaugeChart } from './components/GaugeChart';
import './components/gauge.scss';

/*
 * https://www.chartle.com/
 */
export const Gauge: React.FC<VisualizationProps> = (props) => {
  const { panel, theme, datasets } = props;
  const [ds, setDS] = useState<any>();

  useEffect(() => {
    const datasetsOfGauge = DataSetKit.createStatsDatasets(datasets);
    setDS({
      labels: ['10%', '100%'],
      datasets: [
        { label: 'a', data: [70, 20, 10], backgroundColor: ['green', 'orange', 'red'] },
        { label: 'b', data: [34.34, 90], borderWidth: 1, backgroundColor: ['green', 'rgba(46,50,56, .1)'], weight: 10 },
      ],
    });
  }, [datasets]);
  return <GaugeChart theme={theme} datasets={ds} options={panel.options || {}} panel={panel} />;
};
