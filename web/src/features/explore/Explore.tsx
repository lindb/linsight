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
import { get, set, isEmpty, cloneDeep } from 'lodash-es';
import './explore.scss';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';

const { Header } = Layout;
const DefaultPanel = {
  targets: [{}],
  type: 'timeseries',
  fieldConfig: {
    defaults: {
      unit: 'short',
    },
  },
};

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
    addToChartBtn.current.setOptions(panel);

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
          <AddToDashboard
            btnType="tertiary"
            btnTheme="light"
            getCharts={() => {
              const p = cloneDeep(panel);
              p.title = 'Panel title';
              return [
                {
                  config: p,
                },
              ];
            }}
          />
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
    if (!options || isEmpty(options)) {
      return DefaultPanel;
    }
    try {
      const panel = JSON.parse(options);
      if (!panel) {
        return DefaultPanel;
      }
      if (isEmpty(get(panel, 'targets', []))) {
        set(panel, 'targets', [{}]);
      }
      set(panel, 'type', 'timeseries');
      set(panel, 'fieldConfig', {
        defaults: {
          unit: 'short',
        },
      });
      return panel;
    } catch (err) {
      console.warn('parse metric explore error', err);
    }
    return DefaultPanel;
  };
  return (
    <PanelEditContextProvider initPanel={getOptions('left')}>
      <ExploreContent />
    </PanelEditContextProvider>
  );
};

export default Explore;
