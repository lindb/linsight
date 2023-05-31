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
import { PanelSetting, VisualizationRepositoryInst } from '@src/types';
import React, { useCallback, useContext, useEffect, useState, forwardRef, useImperativeHandle, useRef } from 'react';
import { Icon, LazyLoad, SimpleStatusTip } from '@src/components';
import { PlatformContext } from '@src/contexts';
import { useMetric } from '@src/hooks';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DashboardStore } from '@src/stores';
import classNames from 'classnames';
import { get } from 'lodash-es';
import { ObjectKit } from '@src/utils';

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
                icon={<Icon icon="icon-eye" />}
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
                icon={<Icon icon="icon-explore" />}
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

const Panel: React.FC<{ panel: PanelSetting; shortcutKey?: boolean; isStatic?: boolean; className?: string }> = (
  props
) => {
  const { panel, shortcutKey, isStatic, className } = props;
  const [searchParams] = useSearchParams();
  const { theme } = useContext(PlatformContext);
  const navigate = useNavigate();
  const { loading, error, result } = useMetric(panel?.targets || []);
  const plugin = VisualizationRepositoryInst.get(`${panel.type}`);
  const header = useRef<any>();
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
      return <div>not support</div>;
    }
    const Visualization = plugin.Visualization;
    return <Visualization datasets={result} theme={theme} panel={ObjectKit.merge(plugin.getDefaultOptions(), panel)} />;
  };
  const panelCls = classNames('dashboard-panel', className, {
    'dashboard-panel-static': isStatic,
  });

  return (
    <div
      style={{ width: '100%', height: '100%' }}
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

export default Panel;
