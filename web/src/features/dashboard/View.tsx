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
import * as _ from 'lodash-es';
import { AddPanelWidget, Panel } from '@src/components';
import { Observer } from 'mobx-react-lite';
import { DashboardStore } from '@src/stores';
import { toJS } from 'mobx';
import RGL, { WidthProvider } from 'react-grid-layout';
import { DefaultColumns, DefaultRowHeight } from '@src/constants';
import ViewVariables from './components/ViewVariables';

const ReactGridLayout = WidthProvider(RGL);

const View: React.FC = () => {
  const buildLayout = (panels: any) => {
    const layout: any[] = [];
    toJS(panels || []).map((item: any, _index: number) => {
      if (!item) {
        return;
      }
      item.grid.i = item.id;
      layout.push(item.grid);
    });
    console.log('layout.......', toJS(panels), layout);
    return layout;
  };

  const renderPanel = (panel: any) => {
    if (panel.type === 'addPanel') {
      return <AddPanelWidget panel={panel} />;
    }
    return <Panel panel={panel} shortcutKey />;
  };

  return (
    <>
      <div style={{ margin: '6px 6px 0px 6px' }}>
        <ViewVariables />
      </div>
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
                    layout={buildLayout(DashboardStore.dashboard.config?.panels)}
                    useCSSTransforms={false}
                    onLayoutChange={(layout: any) => {
                      (layout || []).forEach((item: any) => {
                        const panels = _.get(DashboardStore.dashboard, 'config.panels', []);
                        const index = _.findIndex(panels, { id: item.i });
                        DashboardStore.updatePanelConfig(panels[index], { grid: item });
                      });
                    }}
                    margin={[6, 6]}
                    cols={DefaultColumns}
                    rowHeight={DefaultRowHeight}
                    width={width}
                    draggableHandle=".semi-card-header">
                    {DashboardStore.dashboard.config?.panels?.map((item: any, index: number) => {
                      if (!item) {
                        return null;
                      }
                      return <div key={item.id ? `${item.id}` : `${index}`}>{renderPanel(item)}</div>;
                    })}
                  </ReactGridLayout>
                )}
              </Observer>
            </div>
          );
        }}
      </AutoSizer>
    </>
  );
};

export default View;
