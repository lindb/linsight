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
import { Layout } from '@douyinfe/semi-ui';
import React, { useEffect, useRef, useState } from 'react';
import { AddToCharts, AddToDashboard, DatasourceSelectForm, Icon, MetricExplore, TimePicker } from '@src/components';
import { DatasourceInstance } from '@src/types';
import { useSearchParams } from 'react-router-dom';
import { DatasourceStore } from '@src/stores';
import { get } from 'lodash-es';
import { toJS } from 'mobx';
import './explore.scss';

const { Header } = Layout;

const Explore: React.FC = () => {
  const { datasources } = DatasourceStore;
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    return toJS(get(datasources, '[0]'));
  });
  const [searchParams, setSearchParams] = useSearchParams();
  const addToChartBtn = useRef<any>();
  const ds = searchParams.get('ds');

  useEffect(() => {
    if (ds) {
      setDatasource(DatasourceStore.getDatasource(ds));
    }
  }, [ds]);

  return (
    <Layout className="linsight-explore">
      <Header className="linsight-feature-header">
        <div className="explore-header">
          <Icon icon="explore" />
          <DatasourceSelectForm
            noLabel
            value={ds || get(datasources, '[0].setting.uid')}
            style={{ width: 200 }}
            onChange={(instance: DatasourceInstance) => {
              // setSearchParams({ q: JSON.stringify({ datasouce: { uid: instance.setting.uid } }) });
            }}
          />
        </div>
        <div style={{ display: 'flex', gap: 4 }}>
          <AddToCharts ref={addToChartBtn} />
          <AddToDashboard btnType="tertiary" btnTheme="light" />
          <TimePicker />
        </div>
      </Header>
      {datasource && (
        <MetricExplore
          datasource={datasource}
          onValueChange={(values: any) => {
            if (addToChartBtn.current) {
              addToChartBtn.current.setOptions(values);
            }
          }}
        />
      )}
    </Layout>
  );
};

export default Explore;
