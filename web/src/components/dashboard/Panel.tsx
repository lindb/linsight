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
} from '@douyinfe/semi-icons';
import { PanelSetting, Tracker, VisualizationPlugin, VisualizationRepositoryInst } from '@src/types';
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
import { Icon, LazyLoad, SimpleStatusTip } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { useMetric } from '@src/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DashboardStore } from '@src/stores';
import classNames from 'classnames';
import { get, isEmpty, pick } from 'lodash-es';
import { DataSetKit, ObjectKit } from '@src/utils';
import './panel.scss';
import { PanelVisualizationOptions } from '@src/constants';

const { Text } = Typography;

const PanelHeader = forwardRef(
  (
    props: {
      panel: PanelSetting;
      isStatic?: boolean;
      isLoading: boolean;
      error: any;
    },
    ref
  ) => {
    const { panel, isLoading, error, isStatic } = props;
    const [searchParams] = useSearchParams();
    const navigate = useNavigate();
    const [menuVisible, setMenuVisible] = useState(false);
    useImperativeHandle(ref, () => ({
      // panel invoke, reduce panel rerender
      showMenu(show: boolean) {
        setMenuVisible(show);
      },
    }));

    return (
      <div className="panel-header grid-drag-handle">
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
                  searchParams.set('left', JSON.stringify(get(panel, 'targets[0]', {})));
                  searchParams.delete('d');
                  navigate({ pathname: '/explore', search: searchParams.toString() });
                }}>
                Explore
              </Dropdown.Item>
              <Dropdown.Divider />
              {!isStatic && (
                <Dropdown.Item
                  icon={<IconDeleteStroked />}
                  type="danger"
                  onClick={() => DashboardStore.deletePanel(panel)}>
                  Remove
                </Dropdown.Item>
              )}
            </Dropdown.Menu>
          }>
          <IconMenu className="btn" style={{ display: menuVisible ? 'block' : 'none' }} />
        </Dropdown>
      </div>
    );
  }
);
PanelHeader.displayName = 'PanelHeader';

const getPanelOptions = (panel: PanelSetting, plugin: VisualizationPlugin): PanelSetting => {
  return pick(ObjectKit.merge(plugin.getDefaultOptions(), panel), PanelVisualizationOptions);
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
    return DataSetKit.createDatasets(result, datasetType);
  });
  // tracker data/config if changed, opt reduce re-render
  const resultTrackerRef = useRef() as MutableRefObject<Tracker<any>>;
  const panelTrackerRef = useRef() as MutableRefObject<Tracker<any>>;
  /*
   * do initialize logic
   */
  useMemo(() => {
    resultTrackerRef.current = new Tracker<any>(DataSetKit.createDatasets(result, datasetType));
    panelTrackerRef.current = new Tracker<any>(panelCfg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const rs = DataSetKit.createDatasets(result, datasetType);
    if (resultTrackerRef.current.isChanged(rs)) {
      resultTrackerRef.current.setNewVal(rs);
      setDatasets(rs);
    }
  }, [result, datasetType]);

  useEffect(() => {
    const cfg = getPanelOptions(panel, plugin);
    if (panelTrackerRef.current.isChanged(cfg)) {
      panelTrackerRef.current.setNewVal(cfg);
      setPanelCfg(cfg);
    }
  }, [panel, plugin]);

  return <Visualization datasets={datasets} theme={theme} panel={panelCfg} />;
};

const Panel: React.FC<{ panel: PanelSetting; shortcutKey?: boolean; isStatic?: boolean; className?: string }> = (
  props
) => {
  const { panel, shortcutKey, isStatic, className } = props;
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const plugin = VisualizationRepositoryInst.get(`${panel.type}`);
  const header = useRef<any>();
  const { loading, error, result } = useMetric(panel?.targets || []);
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
          navigate({ pathname: '/explore' });
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

  const panelCls = classNames('dashboard-panel', className, {
    'dashboard-panel-static': isStatic,
  });

  return (
    <div
      className="panel"
      onMouseEnter={() => {
        if (shortcutKey) {
          window.addEventListener('keydown', handleKeyDown);
        }
        toggleHeaderMenu(true);
      }}
      onMouseLeave={() => {
        if (shortcutKey) {
          window.removeEventListener('keydown', handleKeyDown);
        }
        toggleHeaderMenu(false);
      }}>
      <LazyLoad>
        <Card
          className={panelCls}
          header={<PanelHeader ref={header} panel={panel} isStatic={isStatic} isLoading={loading} error={error} />}>
          {renderContent()}
        </Card>
      </LazyLoad>
    </div>
  );
};

export default React.memo(Panel);
