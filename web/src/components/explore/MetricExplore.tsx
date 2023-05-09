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
import React, { useState } from 'react';
import { DatasourceInstance } from '@src/types';
import {} from 'lodash-es';
import { Card, Collapse, Typography } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import Panel from '../dashboard/Panel';
const { Text } = Typography;

const MetricExplore: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const [query, setQuery] = useState<any>(null);
  const plugin = datasource.plugin;
  const QueryEditor = plugin.components.QueryEditor;
  if (!QueryEditor) {
    return null;
  }
  return (
    <>
      <Card className="linsight-feature">
        <div className="linsight-explore">
          <Collapse activeKey={['A']} expandIconPosition="left">
            <Collapse.Panel
              header={
                <div style={{ display: 'flex', width: '100%', alignItems: 'center' }}>
                  <div style={{ flex: 1 }}>
                    <Text>A</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <IconCopy />
                  </div>
                </div>
              }
              itemKey="A">
              <QueryEditor datasource={datasource} onChange={(values: object) => setQuery(values)} />
            </Collapse.Panel>
          </Collapse>
        </div>
      </Card>
      <div className="linsight-feature" style={{ marginTop: 0, height: 300 }}>
        <Panel
          panel={{
            type: 'timeseries',
            targets: [{ datasource: { uid: datasource.setting.uid }, request: query }],
          }}
        />
      </div>
    </>
  );
};

export default MetricExplore;
