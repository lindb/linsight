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
import { OrientationType, VisualizationProps } from '@src/types';
import React, { useEffect } from 'react';
import StatChart from './components/StatChart';
import { StatOptions } from './types';
import './stat.scss';

const Stat: React.FC<VisualizationProps> = (props) => {
  const { panel, datasets } = props;
  const options = (panel.options || {}) as StatOptions;
  useEffect(() => {
    // hack: change parent div style
    const parentElements = document.querySelectorAll('.panel-type-stat .semi-card-body') as any;
    (parentElements || []).forEach((p: any) => {
      p.style.padding = '0px';
    });
  }, []);
  return (
    <div
      className="stat-list"
      style={{ flexDirection: options.orientation === OrientationType.horizontal ? 'column' : 'row' }}>
      {(datasets || []).map((ds: any, index: number) => (
        <StatChart key={index} dataset={ds} options={options} />
      ))}
    </div>
  );
};

export default Stat;
