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
import React, { useContext, useEffect, useState } from 'react';
import { DatasourceInstance } from '@src/types';
import { isEmpty, get } from 'lodash-es';
import { Card, Collapse, Typography } from '@douyinfe/semi-ui';
import { IconCopy } from '@douyinfe/semi-icons';
import Panel from '../dashboard/Panel';
import './metric-explore.scss';
import { QueryEditContext, QueryEditContextProvider } from '@src/contexts';
import { useSearchParams } from 'react-router-dom';
const { Text } = Typography;

const MetricQueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;
  const { values } = useContext(QueryEditContext);
  const QueryEditor = plugin.components.QueryEditor;
  const [searchParams, setSearchParams] = useSearchParams();

  useEffect(() => {
    searchParams.set('left', JSON.stringify({ datasource: { uid: datasource.setting.uid }, request: values }));
    setSearchParams(searchParams);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [values, datasource.setting.uid]);

  if (!QueryEditor) {
    return null;
  }
  return <QueryEditor datasource={datasource} />;
};

const MetricExplore: React.FC<{ datasource: DatasourceInstance; onValueChange: (options: any) => void }> = (props) => {
  const { datasource, onValueChange } = props;
  const [searchParams] = useSearchParams();
  const getTargets = (key: string) => {
    const targets = searchParams.get(key);
    if (isEmpty(targets)) {
      return {};
    }
    try {
      return JSON.parse(`${targets}`);
    } catch (err) {
      console.log('parse metric explore error', err);
    }
  };

  const [left, setLeft] = useState(() => {
    return getTargets('left');
  });

  useEffect(() => {
    setLeft(getTargets('left'));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  useEffect(() => {
    onValueChange({
      type: 'timeseries',
      targets: [left],
    });
  }, [left, onValueChange]);

  return (
    <>
      <Card className="linsight-feature">
        <div className="metric-explore">
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
              <QueryEditContextProvider initValues={get(left, 'request', {})}>
                <MetricQueryEditor datasource={datasource} />
              </QueryEditContextProvider>
            </Collapse.Panel>
          </Collapse>
        </div>
      </Card>
      <div className="linsight-feature" style={{ marginTop: 0, height: 300 }}>
        <Panel
          panel={{
            type: 'timeseries',
            targets: [left],
          }}
        />
      </div>
    </>
  );
};

export default MetricExplore;
