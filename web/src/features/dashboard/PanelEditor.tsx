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
import { IconBellStroked } from '@douyinfe/semi-icons';
import SplitPane from 'react-split-pane';
import { PanelSetting as PanelOptions } from '@src/types';
import { DatasourceSelectForm, Notification, Panel } from '@src/components';
import { get, isEmpty } from 'lodash-es';
import { observer } from 'mobx-react-lite';
import { PanelStore, DashboardStore, DatasourceStore } from '@src/stores';
import { toJS } from 'mobx';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DatasourceInstance } from '@src/types';
import PanelSetting from './PanelSetting';
import ViewVariables from './components/ViewVariables';
import { QueryEditContext, QueryEditContextProvider } from '@src/contexts';

const Split: any = SplitPane;

const MetricQueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;
  const { values } = useContext(QueryEditContext);
  const QueryEditor = plugin.components.QueryEditor;
  useEffect(() => {
    DashboardStore.updatePanelConfig(PanelStore.panel, {
      targets: [{ datasource: { uid: datasource.setting.uid }, request: values }],
    });
  }, [values, datasource.setting.uid]);

  if (!QueryEditor) {
    return null;
  }
  return <QueryEditor datasource={datasource} />;
};

const MetricSetting: React.FC<{ panel?: PanelOptions }> = (props) => {
  const { panel } = props;
  const { datasources } = DatasourceStore;
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    return toJS(get(datasources, '[0]'));
  });

  const QueryEditor = () => {
    if (!datasource) {
      return <></>;
    }
    const plugin = datasource.plugin;
    const QueryEditor = plugin.components.QueryEditor;
    if (!QueryEditor) {
      return <></>;
    }
    console.log('panel.....', toJS(panel));
    const targets = get(panel, 'targets', []);
    if (isEmpty(targets)) {
      // no targets, init empty query editor
      return (
        <QueryEditContextProvider>
          <MetricQueryEditor datasource={datasource} />
        </QueryEditContextProvider>
      );
    }
    return (
      <>
        {targets.map((target: any, index: number) => {
          return (
            <QueryEditContextProvider key={index} initValues={get(target, 'request', {})}>
              <MetricQueryEditor datasource={datasource} />
            </QueryEditContextProvider>
          );
        })}
      </>
    );
  };

  return (
    <div>
      <DatasourceSelectForm
        noLabel
        value={get(datasource, 'setting.uid')}
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
  const navigate = useNavigate();
  const panel = DashboardStore.getPanel(parseInt(`${searchParams.get('panel')}`));
  const [size, setSize] = useState<number>(DefaultOptionsEditorSize);
  const initialized = useRef(false);

  useEffect(() => {
    if (!panel) {
      Notification.error('Panel not found');
      searchParams.delete('panel');
      navigate({ pathname: '/dashboard', search: searchParams.toString() });
      return;
    }
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
                      {get(PanelStore.panel, 'targets', []).length}
                    </Avatar>
                  </span>
                }
                itemKey="1">
                <Card bodyStyle={{ padding: 12 }}>
                  <MemoMetricSetting panel={panel} />
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
