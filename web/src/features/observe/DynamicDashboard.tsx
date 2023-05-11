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
import React, { useEffect } from 'react';
import { DashboardSrv } from '@src/services';
import { get, isEmpty } from 'lodash-es';
import { Form } from '@douyinfe/semi-ui';
import AutoSizer from 'react-virtualized-auto-sizer';
import { observer } from 'mobx-react-lite';
import { Panel } from '@src/components';
import { toJS } from 'mobx';
import RGL, { WidthProvider } from 'react-grid-layout';
import { DashboardStore, MenuStore } from '@src/stores';
import { DefaultColumns, DefaultRowHeight } from '@src/constants';
import { useRequest } from '@src/hooks';

const ReactGridLayout = WidthProvider(RGL);

const DynamicDashboard: React.FC = () => {
  const { currentMenu } = MenuStore;
  console.log('DynamicDashboard menu', toJS(currentMenu));
  const dashboardId = get(currentMenu, 'props.dashboard');
  const { result: dashboard, loading } = useRequest(
    ['load-dashboard', dashboardId],
    async () => {
      if (isEmpty(dashboardId)) {
        return null;
      }
      return DashboardSrv.getDashboard(`${dashboardId}`);
    },
    {
      // enabled: !_.isEmpty(dashboardId),
    }
  );

  useEffect(() => {
    if (dashboard) {
      DashboardStore.setDashboard(dashboard);
    }
    return () => {
      return DashboardStore.setDashboard(undefined as any);
    };
  }, [dashboard]);

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

  if (loading) {
    return <div>loading</div>;
  }
  console.log('no dashboard', dashboard);
  if (!dashboard) {
    return <div>dashboard not exit</div>;
  }

  const renderPanel = (panel: any) => {
    return <Panel isStatic panel={panel} shortcutKey />;
  };
  return (
    <>
      <Form style={{ marginLeft: 8 }} className="linsight-form" labelPosition="inset">
        <Form.Select label="Variable" field="test" optionList={[{ label: 'option1', value: 'option1' }]} />
      </Form>
      <AutoSizer disableHeight>
        {({ width }) => {
          if (width == 0) {
            return null;
          }
          return (
            <div style={{ width: `${width}px` }}>
              <ReactGridLayout
                className="layout"
                layout={buildLayout(dashboard.config?.panels)}
                useCSSTransforms={false}
                margin={[6, 6]}
                cols={DefaultColumns}
                rowHeight={DefaultRowHeight}
                width={width}
                isDraggable={false}
                isResizable={false}
                isDroppable={false}>
                {dashboard.config?.panels?.map((item: any, index: number) => {
                  if (!item) {
                    return null;
                  }
                  return <div key={item.id ? `${item.id}` : `${index}`}>{renderPanel(item)}</div>;
                })}
              </ReactGridLayout>
            </div>
          );
        }}
      </AutoSizer>
    </>
  );
};

export default observer(DynamicDashboard);
