import React, { useEffect } from 'react';
import { DashboardSrv } from '@src/services';
import { useQuery } from '@tanstack/react-query';
import * as _ from 'lodash-es';
import { Form } from '@douyinfe/semi-ui';
import AutoSizer from 'react-virtualized-auto-sizer';
import { observer, Observer } from 'mobx-react-lite';
import { Panel } from '@src/components';
import { toJS } from 'mobx';
import RGL, { WidthProvider } from 'react-grid-layout';
import { DashboardStore, MenuStore } from '@src/stores';
import { DefaultColumns, DefaultRowHeight } from '@src/constants';

const ReactGridLayout = WidthProvider(RGL);

const DynamicDashboard: React.FC = () => {
  const { currentMenu } = MenuStore;
  console.log('DynamicDashboard menu', toJS(currentMenu));
  const dashboardId = _.get(currentMenu, 'props.dashboard');
  const { data: dashboard, isLoading } = useQuery(
    ['load-dashboard', dashboardId],
    async () => {
      if (_.isEmpty(dashboardId)) {
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
    toJS(panels || []).map((item: any, index: number) => {
      if (!item) {
        return;
      }
      item.grid.i = item.id;
      layout.push(item.grid);
    });
    console.log('layout.......', toJS(panels), layout);
    return layout;
  };

  if (isLoading) {
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
