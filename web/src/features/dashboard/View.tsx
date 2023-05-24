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
import React from 'react';
import AutoSizer from 'react-virtualized-auto-sizer';
import { get, findIndex } from 'lodash-es';
import { AddPanelWidget, Panel } from '@src/components';
import { Observer } from 'mobx-react-lite';
import { DashboardStore } from '@src/stores';
import { toJS } from 'mobx';
import RGL, { WidthProvider } from 'react-grid-layout';
import { DefaultColumns, DefaultRowHeight, RowPanelType, VisualizationAddPanelType } from '@src/constants';
import ViewVariables from './components/ViewVariables';
import { VariableContextProvider } from '@src/contexts';
import { Variable } from '@src/types';
import RowPanel from './components/RowPanel';

const ReactGridLayout = WidthProvider(RGL);

const View: React.FC = () => {
  const { dashboard } = DashboardStore;
  const variables: Variable[] = get(dashboard, 'config.variables', []);

  const buildLayout = (panels: any) => {
    const layout: any[] = [];
    toJS(panels || []).map((item: any, _index: number) => {
      if (!item || get(item, '_hidden', false)) {
        return;
      }
      if (item.type === RowPanelType) {
        item.grid.isResizable = false;
      }
      item.grid.i = item.id;
      layout.push(item.grid);
    });
    return layout;
  };

  const renderPanel = (panel: any) => {
    switch (panel.type) {
      case VisualizationAddPanelType:
        return <AddPanelWidget panel={panel} />;
      case RowPanelType:
        return <RowPanel panel={panel} />;
      default:
        return <Panel panel={panel} shortcutKey />;
    }
  };

  const renderPanels = () => {
    const panels = DashboardStore.getPanels();
    return panels.map((item: any, index: number) => {
      if (!item || get(item, '_hidden', false)) {
        return null;
      }
      return <div key={item.id ? `${item.id}` : `${index}`}>{renderPanel(item)}</div>;
    });
  };

  return (
    <VariableContextProvider variables={variables}>
      <ViewVariables className="variables" />
      <AutoSizer disableHeight>
        {({ width }) => {
          if (width == 0) {
            return null;
          }
          return (
            <div style={{ width: `${width}px` }}>
              <Observer>
                {() => (
                  <ReactGridLayout
                    className="layout"
                    layout={buildLayout(DashboardStore.getPanels())}
                    useCSSTransforms={false}
                    onLayoutChange={(layout: any) => {
                      (layout || []).forEach((item: any) => {
                        const panels = get(DashboardStore.dashboard, 'config.panels', []);
                        const index = findIndex(panels, { id: item.i });
                        DashboardStore.updatePanelConfig(panels[index], { grid: item });
                        DashboardStore.sortPanels();
                      });
                    }}
                    margin={[6, 6]}
                    cols={DefaultColumns}
                    rowHeight={DefaultRowHeight}
                    width={width}
                    draggableHandle=".grid-drag-handle">
                    {renderPanels()}
                  </ReactGridLayout>
                )}
              </Observer>
            </div>
          );
        }}
      </AutoSizer>
    </VariableContextProvider>
  );
};

export default View;
