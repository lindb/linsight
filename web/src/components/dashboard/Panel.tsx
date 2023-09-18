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
import { Card, Typography, Dropdown, Popover } from '@douyinfe/semi-ui';
import {
  IconLineChartStroked,
  IconEdit2Stroked,
  IconDeleteStroked,
  IconCopyStroked,
  IconMenu,
  IconUnlink,
} from '@douyinfe/semi-icons';
import { DataSetType, PanelSetting, Tracker, VisualizationPlugin, VisualizationRepositoryInst } from '@src/types';
import React, {
  useCallback,
  useContext,
  useEffect,
  useState,
  forwardRef,
  useImperativeHandle,
  useRef,
  MutableRefObject,
  useMemo,
} from 'react';
import { Icon, LazyLoad, SimpleStatusTip, AddToCharts, UnlinkChart } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { useMetric, useRequest } from '@src/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DashboardStore } from '@src/stores';
import classNames from 'classnames';
import { get, isEmpty, pick, cloneDeep, isArray, omit, has, unset } from 'lodash-es';
import { ChartKit, DataSetKit, ObjectKit } from '@src/utils';
import './panel.scss';
import { ChartPropsKey, PanelVisualizationOptions } from '@src/constants';
import { ChartSrv } from '@src/services';
import { toJS } from 'mobx';

const { Text } = Typography;

const PanelHeader = forwardRef(
  (
    props: {
      panel: PanelSetting;
      isStatic?: boolean;
      isLoading: boolean;
      menu?: boolean;
      error: any;
    },
    ref
  ) => {
    const { panel, isLoading, error, isStatic, menu } = props;
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const [menuVisible, setMenuVisible] = useState(false);
    const [addChartVisible, setAddChartVisible] = useState(false);
    const [unlinkVisible, setUnlinkVisible] = useState(false);
    const addToChartBtn = useRef<any>();
    useImperativeHandle(ref, () => ({
      // panel invoke, reduce panel rerender
      showMenu(show: boolean) {
        setMenuVisible(show);
      },
    }));

    const renderChartRepo = () => {
      if (has(panel, ChartPropsKey)) {
        // unlink
        return (
          <Dropdown.Item icon={<IconUnlink />} onClick={() => setUnlinkVisible(true)}>
            Unlink chart
          </Dropdown.Item>
        );
      }
      // link
      return (
        <Dropdown.Item
          icon={<Icon icon="repo" />}
          onClick={() => {
            const chartCfg = cloneDeep(panel);
            // init integration if has dashboard
            chartCfg.integration = DashboardStore.dashboard?.integration;
            addToChartBtn.current.setOptions(chartCfg);
            setAddChartVisible(true);
          }}>
          Save as chart
        </Dropdown.Item>
      );
    };

    return (
      <div className="panel-header grid-drag-handle">
        <UnlinkChart
          visible={unlinkVisible}
          setVisible={setUnlinkVisible}
          unlinkChart={() => {
            const chartCfg = cloneDeep(panel);
            unset(chartCfg, 'libraryPanel');
            DashboardStore.updatePanel(chartCfg);
          }}
        />
        <div className="title">
          {panel.title && (
            <>
              <IconLineChartStroked />
              {panel.description ? (
                <Popover showArrow arrowPointAtCenter content={<article>{panel.description}</article>}>
                  <Text>{panel.title}</Text>
                </Popover>
              ) : (
                <Text>{panel.title}</Text>
              )}
            </>
          )}
        </div>
        <SimpleStatusTip isLoading={isLoading} error={error} />
        {menu && (
          <Dropdown
            render={
              <Dropdown.Menu>
                <Dropdown.Item
                  icon={<Icon icon="eye" />}
                  onClick={() => {
                    searchParams.set('panel', `${panel.id}`);
                    navigate({ pathname: '/dashboard/panel/view', search: searchParams.toString() });
                  }}>
                  View
                </Dropdown.Item>
                {!isStatic && (
                  <Dropdown.Item
                    icon={<IconEdit2Stroked />}
                    onClick={() => {
                      searchParams.set('panel', `${panel.id}`);
                      navigate({ pathname: '/dashboard/panel/edit', search: searchParams.toString() });
                    }}>
                    Edit
                  </Dropdown.Item>
                )}
                {!isStatic && (
                  <Dropdown.Item icon={<IconCopyStroked />} onClick={() => DashboardStore.clonePanel(panel)}>
                    Clone
                  </Dropdown.Item>
                )}
                <Dropdown.Item
                  icon={<Icon icon="explore" />}
                  onClick={() => {
                    searchParams.set('left', JSON.stringify(pick(panel, ['datasource', 'targets'])));
                    navigate({ pathname: '/explore', search: searchParams.toString() });
                  }}>
                  Explore
                </Dropdown.Item>
                {!isStatic && (
                  <>
                    {renderChartRepo()}
                    <Dropdown.Divider />
                    <Dropdown.Item
                      icon={<IconDeleteStroked />}
                      type="danger"
                      onClick={() => DashboardStore.deletePanel(panel)}>
                      Remove
                    </Dropdown.Item>
                  </>
                )}
              </Dropdown.Menu>
            }>
            <IconMenu className="btn" style={{ display: menuVisible ? 'block' : 'none' }} />
          </Dropdown>
        )}
        <AddToCharts
          ref={addToChartBtn}
          link={(chart: any) => {
            DashboardStore.updatePanelConfig(panel, {
              libraryPanel: {
                uid: chart.uid,
                name: chart.title,
              },
            });
          }}
          visible={addChartVisible}
          setVisible={setAddChartVisible}
        />
      </div>
    );
  }
);
PanelHeader.displayName = 'PanelHeader';

const getPanelOptions = (panel: PanelSetting, plugin: VisualizationPlugin): PanelSetting => {
  return cloneDeep(pick(ObjectKit.merge(plugin.getDefaultOptions(), panel), PanelVisualizationOptions));
};

const PanelVisualization: React.FC<{ panel: PanelSetting; plugin: VisualizationPlugin; result: any }> = (props) => {
  const { panel, plugin, result } = props;
  const { theme } = useContext(PlatformContext);
  const Visualization = plugin.Visualization;
  const datasetType = plugin.getDataSetType(panel);
  const [panelCfg, setPanelCfg] = useState<PanelSetting>(() => {
    return getPanelOptions(panel, plugin);
  });
  const [datasets, setDatasets] = useState<any>(() => {
    return DataSetKit.createDatasets(result, datasetType, panel);
  });
  // tracker data/config if changed, opt reduce re-render
  const resultTrackerRef = useRef() as MutableRefObject<Tracker<any>>;
  const panelTrackerRef = useRef() as MutableRefObject<Tracker<any>>;
  /*
   * do initialize logic
   */
  useMemo(() => {
    resultTrackerRef.current = new Tracker<any>(DataSetKit.createDatasets(result, datasetType, panel));
    panelTrackerRef.current = new Tracker<any>(panelCfg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const cfg = getPanelOptions(panel, plugin);
    if (panelTrackerRef.current.isChanged(cfg)) {
      panelTrackerRef.current.setNewVal(cfg);
      setPanelCfg(cfg);
    }
    const rs = DataSetKit.createDatasets(result, datasetType, panel);
    if (resultTrackerRef.current.isChanged(rs)) {
      resultTrackerRef.current.setNewVal(rs);
      setDatasets(rs);
    }
  }, [panel, plugin, datasetType, result]);

  const getDatasets = () => {
    // fix when change visualizatin type, throw error
    // exmaple: timeseries datasets is object, single stat datasets need array
    if (datasetType === DataSetType.SingleStat) {
      if (isArray(datasets)) {
        return datasets;
      } else {
        return [];
      }
    }
    return datasets;
  };
  return <Visualization datasets={getDatasets()} theme={theme} panel={panelCfg} />;
};

const ViewPanel: React.FC<{
  panel: PanelSetting;
  shortcutKey?: boolean;
  isStatic?: boolean;
  className?: string;
  menu?: boolean;
}> = (props) => {
  const { panel, shortcutKey, isStatic, className, menu } = props;
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  console.error(toJS(panel));
  // FIXME: maybe plugin not exist
  const plugin = VisualizationRepositoryInst.get(`${panel.type}`);
  const datasetType = plugin.getDataSetType(panel);
  const header = useRef<any>();
  const container = useRef<any>();
  const { loading, error, result } = useMetric(panel?.targets || [], datasetType, get(panel, 'datasource.uid', ''));

  const [datasets, setDatasets] = useState<any>(result);
  useEffect(() => {
    if (!loading) {
      setDatasets(result);
    }
  }, [loading, result]);

  const handleKeyDown = useCallback(
    (e: any) => {
      if (!e.ctrlKey) {
        return;
      }
      switch (e.keyCode) {
        case 86: // ctrl+v
          searchParams.set('panel', `${panel.id}`);
          navigate({ pathname: '/dashboard/panel/view', search: searchParams.toString() });
          return;
        case 69: // ctrl+e
          searchParams.set('panel', `${panel.id}`);
          navigate({ pathname: '/dashboard/panel/edit', search: searchParams.toString() });
          return;
        case 67: // ctrl+c
          DashboardStore.clonePanel(panel);
          return;
        case 80: // ctrl+p
          searchParams.set('left', JSON.stringify(pick(panel, ['datasource', 'targets'])));
          navigate({ pathname: '/explore', search: searchParams.toString() });
          return;
        case 82: // ctrl+r
          DashboardStore.deletePanel(panel);
          return;
        default:
          break;
      }
    },
    [panel, searchParams, navigate]
  );

  useEffect(() => {
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [handleKeyDown]);

  const toggleHeaderMenu = (show: boolean) => {
    if (header.current) {
      header.current.showMenu(show);
    }
  };

  const renderContent = () => {
    if (!plugin) {
      return (
        <div className="msg">
          <span className="error-msg">Not support {panel.type} visualization</span>
        </div>
      );
    }
    if (isEmpty(datasets)) {
      return (
        <div className="msg">
          <span className="empty-msg">No data</span>
        </div>
      );
    }
    return <PanelVisualization panel={panel} plugin={plugin} result={datasets} />;
  };

  const panelCls = classNames('dashboard-panel', className, `panel-type-${panel.type}`, {
    'dashboard-panel-static': isStatic,
  });

  /*
   * maybe mouse already in panel container, so cannot trigger onMouseEnter
   */
  useEffect(() => {
    const handleMouseMove = (event: any) => {
      const { left, top, width, height } = container.current.getBoundingClientRect();
      const x = event.clientX;
      const y = event.clientY;
      const isInside = x >= left && x <= left + width && y >= top && y <= top + height;
      if (isInside) {
        toggleHeaderMenu(true);
      }
    };
    document.addEventListener('mousemove', handleMouseMove);
    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
    };
  }, []);

  return (
    <div
      style={{ height: '100%' }}
      ref={container}
      onMouseEnter={() => {
        if (shortcutKey) {
          window.addEventListener('keydown', handleKeyDown);
        }
      }}
      onMouseLeave={() => {
        if (shortcutKey) {
          window.removeEventListener('keydown', handleKeyDown);
        }
        toggleHeaderMenu(false);
      }}>
      <Card
        className={panelCls}
        header={
          <PanelHeader ref={header} panel={panel} isStatic={isStatic} isLoading={loading} error={error} menu={menu} />
        }>
        {renderContent()}
      </Card>
    </div>
  );
};

const ViewChartPanel: React.FC<{
  panel: PanelSetting;
  shortcutKey?: boolean;
  isStatic?: boolean;
  className?: string;
  menu?: boolean;
}> = (props) => {
  const { panel } = props;
  const chartUID = get(panel, ChartPropsKey, '');
  const { result, loading } = useRequest(['load_chart_by_uid', chartUID], () => {
    return ChartSrv.getChart(chartUID);
  });

  if (loading) {
    // TODO: add loading msg??
    return null;
  }

  return <ViewPanel panel={ObjectKit.merge(ChartKit.getChartConfig(result), panel)} {...omit(props, 'panel')} />;
};

const Panel: React.FC<{
  panel: PanelSetting;
  shortcutKey?: boolean;
  isStatic?: boolean;
  className?: string;
  menu?: boolean;
}> = (props) => {
  const { panel } = props;
  const renderContent = () => {
    const chartUID = get(panel, ChartPropsKey, '');
    if (isEmpty(chartUID)) {
      return <ViewPanel {...props} />;
    }
    return <ViewChartPanel {...props} />;
  };

  return (
    <div className="panel">
      <LazyLoad>{renderContent()}</LazyLoad>
    </div>
  );
};

export default React.memo(Panel);
