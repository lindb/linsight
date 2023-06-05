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
import React, { MutableRefObject, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { AddToCharts, AddToDashboard, DatasourceSelectForm, Icon, MetricExplore, TimePicker } from '@src/components';
import { DatasourceInstance, PanelSetting, Tracker } from '@src/types';
import { useSearchParams } from 'react-router-dom';
import { DatasourceStore } from '@src/stores';
import { get, isEmpty } from 'lodash-es';
import './explore.scss';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';

const { Header } = Layout;

const ExploreContent: React.FC = () => {
  const { datasources } = DatasourceStore;
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const [searchParams, setSearchParams] = useSearchParams();
  const addToChartBtn = useRef<any>();
  const getDatasource = () => {
    const datasourceUID = get(panel, 'datasource.uid', get(datasources, '[0].setting.uid'));
    return DatasourceStore.getDatasource(`${datasourceUID}`);
  };
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    return getDatasource();
  });
  const panelTracker = useRef<Tracker<PanelSetting>>() as MutableRefObject<Tracker<PanelSetting>>;

  useMemo(() => {
    panelTracker.current = new Tracker(panel);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // just init

  useEffect(() => {
    if (panelTracker.current.isChanged(panel)) {
      panelTracker.current.setNewVal(panel);
      searchParams.set('left', JSON.stringify(panel));
      setSearchParams(searchParams);
      setDatasource(getDatasource());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [panel]);

  return (
    <Layout className="linsight-explore">
      <Header className="linsight-feature-header">
        <div className="explore-header">
          <Icon icon="explore" />
          <DatasourceSelectForm
            noLabel
            value={datasource?.setting.uid}
            style={{ width: 200 }}
            onChange={(instance: DatasourceInstance) => {
              modifyPanel({ datasource: { uid: instance.setting.uid } });
            }}
          />
        </div>
        <div style={{ display: 'flex', gap: 4 }}>
          <AddToCharts ref={addToChartBtn} />
          <AddToDashboard btnType="tertiary" btnTheme="light" />
          <TimePicker />
        </div>
      </Header>
      {datasource && <MetricExplore datasource={datasource} />}
    </Layout>
  );
};

const Explore: React.FC = () => {
  const [searchParams] = useSearchParams();
  const getOptions = (key: string) => {
    const options = `${searchParams.get(key)}`;
    if (isEmpty(options)) {
      return {};
    }
    try {
      return JSON.parse(options);
    } catch (err) {
      console.warn('parse metric explore error', err);
    }
    return [];
  };
  const panelOptions = getOptions('left');
  return (
    <PanelEditContextProvider initPanel={panelOptions}>
      <ExploreContent />
    </PanelEditContextProvider>
  );
};

export default Explore;
