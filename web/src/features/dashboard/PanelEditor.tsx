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
import React, { useContext, useEffect, useRef, useState } from 'react';
import { Card, Tabs, TabPane, Avatar } from '@douyinfe/semi-ui';
import SplitPane from 'react-split-pane';
import { PanelSetting as PanelOptions } from '@src/types';
import { DatasourceSelectForm, Notification, Panel, QueryEditor } from '@src/components';
import { get, isEmpty } from 'lodash-es';
import { DashboardStore, DatasourceStore } from '@src/stores';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DatasourceInstance } from '@src/types';
import ViewVariables from './components/ViewVariables';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import PanelSetting from './PanelSetting';
import { ChartPropsKey, DefaultVisualizationType, VisualizationAddPanelType } from '@src/constants';
import { ChartSrv } from '@src/services';
import { ChartKit, ObjectKit } from '@src/utils';

const Split: any = SplitPane;
const DefaultOptionsEditorSize = 350;

const MetricSetting: React.FC = () => {
  const { modifyPanel, panel } = useContext(PanelEditContext);
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    const datasourceUID = get(panel, 'datasource.uid', get(DatasourceStore.getDefaultDatasource(), 'setting.uid'));
    return DatasourceStore.getDatasource(`${datasourceUID}`);
  });

  useEffect(() => {
    modifyPanel({
      datasource: { uid: get(datasource, 'setting.uid', ''), type: get(datasource, 'setting.type', '') },
    });
  }, [datasource, modifyPanel]);

  return (
    <div>
      <DatasourceSelectForm
        noLabel
        value={get(datasource, 'setting.uid')}
        style={{ width: 200 }}
        includeMixed
        onChange={(instance: DatasourceInstance) => {
          setDatasource(instance);
        }}
      />
      {datasource && <QueryEditor datasource={datasource} />}
    </div>
  );
};

const MemoMetricSetting = React.memo(MetricSetting);

const Editor: React.FC<{ initSize: number }> = (props) => {
  const { initSize } = props;
  const { panel } = useContext(PanelEditContext);
  const [size, setSize] = useState<number>(initSize);
  return (
    <Split
      onDragFinished={(size: number) => {
        setSize(size);
      }}
      split="vertical"
      size={size}
      primary="second"
      minSize={250}
      maxSize={500}
      style={{ position: 'relative' }}>
      <Split split="horizontal" minSize={340} style={{ overflow: 'auto' }}>
        <div style={{ display: 'flex', flexDirection: 'column', width: '100%' }}>
          <div style={{ marginBottom: 6 }}>
            <ViewVariables />
          </div>
          <Panel panel={panel} />
        </div>
        <Card bodyStyle={{ padding: '8px 20px 12px 20px' }}>
          <Tabs size="small">
            <TabPane
              tab={
                <span>
                  <i className="iconfont icondatabase" style={{ marginRight: 8 }} />
                  Query
                  <Avatar size="extra-extra-small" style={{ margin: 4 }} alt="User">
                    {get(panel, 'targets', []).length}
                  </Avatar>
                </span>
              }
              itemKey="1">
              <MemoMetricSetting />
            </TabPane>
          </Tabs>
        </Card>
      </Split>
      <div style={{ height: '100%', overflow: 'auto' }}>
        <Card bodyStyle={{ padding: 0 }}>
          <PanelSetting />
        </Card>
      </div>
    </Split>
  );
};

const EditPanel: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [size, setSize] = useState<number>(DefaultOptionsEditorSize);
  const [panel, setPanel] = useState<PanelOptions>();
  const initialized = useRef(false);

  useEffect(() => {
    const initPanel = async () => {
      const panel = DashboardStore.getPanel(parseInt(`${searchParams.get('panel')}`));
      if (!panel) {
        // if panel not exist, forward to dashboard page.
        Notification.error('Panel not found');
        searchParams.delete('panel');
        navigate({ pathname: '/dashboard', search: searchParams.toString() });
        return;
      }
      if (panel.type == VisualizationAddPanelType) {
        // if panel is init add widget panel, set init panel title and type
        DashboardStore.updatePanelConfig(panel, {
          title: 'Panel title',
          type: DefaultVisualizationType,
          targets: [{}],
        });
      }
      const chartUID = get(panel, ChartPropsKey, '');
      if (isEmpty(chartUID)) {
        setPanel(panel);
      } else {
        const chart = await ChartSrv.getChart(chartUID);
        setPanel(ObjectKit.merge(ChartKit.getChartConfig(chart), panel));
      }
    };
    initPanel();
  }, [navigate, searchParams]);

  if (!panel) {
    return null;
  }

  return (
    <div
      ref={(container) => {
        if (container && !initialized.current) {
          initialized.current = true;
          setSize(DefaultOptionsEditorSize);
        }
      }}
      className="linsight-feature panel-editor">
      <PanelEditContextProvider initPanel={panel}>
        <Editor initSize={size} />
      </PanelEditContextProvider>
    </div>
  );
};

export default EditPanel;
