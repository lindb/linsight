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
import { Button, Card, Empty, Layout } from '@douyinfe/semi-ui';
import React, { useContext, useRef, useState } from 'react';
import { AddToCharts, AddToDashboard, DatasourceSelectForm, Icon, DataExplore, TimePicker } from '@src/components';
import { DatasourceInstance } from '@src/types';
import { DatasourceStore } from '@src/stores';
import { get, isEmpty, cloneDeep } from 'lodash-es';
import './explore.scss';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import { DatasourceKit } from '@src/utils';

const { Header } = Layout;

const ExploreContent: React.FC = () => {
  const defaultDatasource = DatasourceStore.getDefaultDatasource();
  const [visible, setVisible] = useState(false);
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const addToChartBtn = useRef<any>();
  const getDatasource = () => {
    const datasourceUID = get(panel, 'datasource.uid', get(defaultDatasource, 'setting.uid'));
    return DatasourceStore.getDatasource(`${datasourceUID}`);
  };
  const datasource = getDatasource();

  const renderContent = () => {
    if (!datasource || isEmpty(DatasourceStore.getDatasources(false))) {
      return (
        <Card
          className="linsight-feature"
          bodyStyle={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Empty
            image={<Icon icon="empty" style={{ fontSize: 48 }} />}
            description="No datasource, please add your first datasource."
          />
        </Card>
      );
    }
    return <DataExplore datasource={datasource} />;
  };

  return (
    <Layout className="linsight-explore">
      <Header className="linsight-feature-header">
        <div className="explore-header">
          <Icon icon="explore" />
          <DatasourceSelectForm
            noLabel
            value={datasource?.setting.uid}
            style={{ width: 200 }}
            includeMixed
            onChange={(instance: DatasourceInstance) => {
              modifyPanel({ datasource: { uid: instance.setting.uid, type: instance.plugin.Type } });
            }}
          />
        </div>
        {DatasourceKit.isMetric(datasource) && (
          <div style={{ display: 'flex', gap: 4 }}>
            <AddToCharts ref={addToChartBtn} visible={visible} setVisible={setVisible} />
            <Button type="tertiary" onClick={() => setVisible(true)} icon={<Icon icon="repo" />}>
              Save as chart
            </Button>
            <AddToDashboard
              btnType="tertiary"
              btnTheme="light"
              getCharts={() => {
                const p = cloneDeep(panel);
                p.title = 'Panel title';
                return [
                  {
                    model: p,
                  },
                ];
              }}
            />
            <TimePicker />
          </div>
        )}
      </Header>
      {renderContent()}
    </Layout>
  );
};

const Explore: React.FC = () => {
  return (
    <PanelEditContextProvider initPanel={{}} urlBind="left">
      <ExploreContent />
    </PanelEditContextProvider>
  );
};

export default Explore;
