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
import React, { useEffect, useState } from 'react';
import { VisualizationProps } from '@src/types';
import { TimeSeriesChart } from './components/TimeSeriesChart';
import { DataSetKit } from '@src/utils';
import './components/timeseries.scss';

/**
 * TimeSeries is a visualization component for time series chart.
 */
export const TimeSeries: React.FC<VisualizationProps> = (props) => {
  const { panel, theme, datasets } = props;
  const [ds, setDS] = useState();

  useEffect(() => {
    setDS(DataSetKit.createTimeSeriesDatasets(datasets));
  }, [datasets]);

  return <TimeSeriesChart theme={theme} panel={panel} datasets={ds} options={panel.options} />;
};
