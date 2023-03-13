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
import React, { useEffect, useRef, useState } from 'react';
import { Card, Tabs, TabPane, Avatar } from '@douyinfe/semi-ui';
import { IconBellStroked } from '@douyinfe/semi-icons';
import SplitPane from 'react-split-pane';
import PanelSetting from '@src/features/dashboard/PanelSetting';
import { DatasourceSelect, Panel } from '@src/components';
import * as _ from 'lodash-es';
import { observer } from 'mobx-react-lite';
import { PanelStore, DashboardStore, DatasourceStore } from '@src/stores';
import { toJS } from 'mobx';
import { useSearchParams } from 'react-router-dom';
import { DatasourceInstance } from '@src/types';

const Split: any = SplitPane;

const MetricSetting: React.FC = () => {
  const { datasources } = DatasourceStore;
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    return toJS(_.get(datasources, '[0]'));
  });

  const QueryEditor = () => {
    if (!datasource) {
      return null;
    }
    const plugin = datasource.plugin;
    const QueryEditor = plugin.components.QueryEditor;
    if (!QueryEditor) {
      return null;
    }
    return (
      <QueryEditor
        datasource={datasource}
        onChange={(values) => {
          DashboardStore.updatePanelConfig(PanelStore.panel, {
            targets: [{ datasource: { uid: datasource.setting.uid }, request: values }],
          });
        }}
      />
    );
  };

  return (
    <div>
      <DatasourceSelect
        noLabel
        value={_.get(datasource, 'setting.uid')}
        style={{ width: 200 }}
        onChange={(instance: DatasourceInstance) => {
          setDatasource(instance);
        }}
      />
      <QueryEditor />
    </div>
  );
};

const MemoMetricSetting = React.memo(MetricSetting);
const DefaultOptionsEditorSize = 350;

const EditPanel: React.FC = () => {
  const [searchParams] = useSearchParams();
  const panel = DashboardStore.getPanel(searchParams.get('panel'));
  const [size, setSize] = useState<number>(DefaultOptionsEditorSize);
  const initialized = useRef(false);

  useEffect(() => {
    PanelStore.setPanel(panel);
    return () => {
      if (PanelStore.panel) {
        // need update panel of dashboard when destory panel editor,
        // because panel of PanelStore is a copy from dashboard.
        DashboardStore.updatePanel(PanelStore.panel);
        // reset current edit panel
        PanelStore.setPanel(undefined);
      }
    };
  }, [panel]);

  if (!PanelStore.panel) {
    // FIXME: add no page
    return <div>panel not exit</div>;
  }

  return (
    <div
      ref={(container) => {
        if (container && !initialized.current) {
          initialized.current = true;
          setSize(DefaultOptionsEditorSize);
        }
      }}
      className="linsight-feature"
      style={{ display: 'flex', height: '100vh' }}>
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
        <Split split="horizontal" minSize={260} style={{ overflow: 'auto' }}>
          <Panel panel={PanelStore.panel} />
          <Card bodyStyle={{ padding: '8px 20px 12px 20px' }}>
            <Tabs size="small">
              <TabPane
                tab={
                  <span>
                    <i className="iconfont icondatabase" style={{ marginRight: 8 }} />
                    Query
                    <Avatar size="extra-extra-small" style={{ margin: 4 }} alt="User">
                      {_.get(PanelStore.panel, 'targets', []).length}
                    </Avatar>
                  </span>
                }
                itemKey="1">
                <Card bodyStyle={{ padding: 12 }}>
                  <MemoMetricSetting />
                </Card>
              </TabPane>
              <TabPane
                tab={
                  <span>
                    <IconBellStroked />
                    Alert
                  </span>
                }
                itemKey="2"></TabPane>
            </Tabs>
          </Card>
        </Split>
        <div style={{ height: '100%', overflow: 'auto' }}>
          <Card bodyStyle={{ padding: 0 }}>
            <PanelSetting />
          </Card>
        </div>
      </Split>
    </div>
  );
};

export default observer(EditPanel);
