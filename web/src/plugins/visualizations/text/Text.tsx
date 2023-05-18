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
import { Descriptions, Typography } from '@douyinfe/semi-ui';
import { FormatRepositoryInst, VisualizationProps } from '@src/types';
import { DataSetKit } from '@src/utils';
import React, { useEffect, useState } from 'react';
import { get } from 'lodash-es';

/**
 * Text is a visualization component for text stats.
 */
const Text: React.FC<VisualizationProps> = (props) => {
  const { panel, datasets } = props;
  const [data, setData] = useState();

  useEffect(() => {
    const ds = DataSetKit.createStatsDatasets(datasets);
    const unit = get(panel.options, 'unit');
    const decimals = get(panel.options, 'decimals');
    const newData = (ds?.labels || []).map((label: string, index: number) => {
      return {
        key: label,
        value: (
          <Typography.Text type="success">
            {FormatRepositoryInst.formatString(unit, ds.datasets[index], decimals)}
          </Typography.Text>
        ),
      };
    });
    setData(newData);
  }, [datasets, panel]);
  return <Descriptions align="center" data={data} />;
};

export default Text;
