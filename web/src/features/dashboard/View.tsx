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
import { pick } from 'lodash-es';
import { AddPanelWidget, Panel } from '@src/components';
import { Observer } from 'mobx-react-lite';
import { DashboardStore } from '@src/stores';
import RGL, { WidthProvider } from 'react-grid-layout';
import {
  DefaultColumns,
  DefaultRowHeight,
  PanelGridPos,
  RowPanelType,
  VisualizationAddPanelType,
} from '@src/constants';
import ViewVariables from './components/ViewVariables';
import { PanelSetting } from '@src/types';
import RowPanel from './components/RowPanel';

const ReactGridLayout = WidthProvider(RGL);

const View: React.FC = () => {
  const buildLayout = (panels: any) => {
    const layout: any[] = [];
    (panels || []).map((item: any, _index: number) => {
      if (!item) {
        return;
      }
      const gridPos = pick(item.gridPos, ['i', ...PanelGridPos]) as ReactGridLayout.Layout;
      if (item.type === RowPanelType) {
        gridPos.isResizable = false;
      }
      layout.push(gridPos);
    });
    return layout;
  };

  const renderPanel = (panel: PanelSetting) => {
    switch (panel.type) {
      case VisualizationAddPanelType:
        return <AddPanelWidget panel={panel} />;
      case RowPanelType:
        return <RowPanel key={`${panel.id}`} panel={panel} />;
      default:
        return <Panel key={`${panel.id}`} panel={panel} shortcutKey menu />;
    }
  };

  const renderPanels = () => {
    const panels = DashboardStore.getPanels();
    return panels.map((item: PanelSetting, _index: number) => {
      if (!item) {
        return null;
      }
      return <div key={item.id}>{renderPanel(item)}</div>;
    });
  };

  const updatePanelGridPos = (layout: ReactGridLayout.Layout[]) => {
    // NOTE: if use onLayoutChange will re-layout when component init
    // and impact lazy load when toggle row
    (layout || []).forEach((item: any) => {
      const panel = DashboardStore.getPanel(parseInt(item.i));
      if (panel) {
        DashboardStore.updatePanelConfig(panel, { gridPos: item });
        DashboardStore.sortPanels();
      }
    });
  };

  return (
    <div>
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
                    onResizeStop={updatePanelGridPos}
                    onDragStop={updatePanelGridPos}
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
    </div>
  );
};

export default View;
