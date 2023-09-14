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
import { Button, Dropdown, Layout, Typography } from '@douyinfe/semi-ui';
import { IconInheritStroked, IconDescend, IconFlowChartStroked, IconPieChart2Stroked } from '@douyinfe/semi-icons';
import { useTrace } from '@src/hooks';
import React from 'react';
import TimelineView from './timeline/TimelineView';
import { Query } from '@src/types';
import FlameView from './flame/FlameView';
import { cloneDeep, isEmpty } from 'lodash-es';
import { Icon, Loading, StatusTip } from '@src/components';
import './trace-view.scss';
import { createSearchParams, useSearchParams } from 'react-router-dom';
import SpanHeader from './components/SpanHeader';

const { Text } = Typography;
const { Header } = Layout;

const TraceView: React.FC<{
  traceId?: string;
  spanId?: string;
  datasources?: string[];
  openSelectedSpan?: boolean;
  newWindowLink?: boolean;
}> = (props) => {
  const { traceId, spanId, datasources, openSelectedSpan, newWindowLink } = props;
  const [searchParams, setSearchParams] = useSearchParams();
  const viewType = searchParams.get('view') || 'timeline';

  const buildQueries = (): Query[] => {
    return (datasources || []).map((ds: string) => {
      return {
        datasource: { uid: ds },
        request: {
          traceId: traceId,
        },
      };
    });
  };
  const { loading, traces, tree, spanMap } = useTrace(buildQueries());

  // FIXME: add trace/datasouce empty check
  if (loading) {
    return (
      <div className="loading">
        <Loading />
      </div>
    );
  }

  if (isEmpty(traces) || isEmpty(tree)) {
    // FIXME: no found page
    return (
      <div className="loading">
        <StatusTip isEmpty />
      </div>
    );
  }

  const renderContent = () => {
    switch (viewType) {
      case 'flame':
        return <FlameView traces={cloneDeep(traces)} />;
      default:
        return <TimelineView traceTree={tree} spanId={spanId} spanMap={spanMap} openSelectedSpan={openSelectedSpan} />;
    }
  };

  const renderIcon = () => {
    switch (viewType) {
      case 'flame':
        return <IconDescend />;
      default:
        return <IconInheritStroked />;
    }
  };

  const changeViewType = (type: string) => {
    searchParams.set('view', type);
    setSearchParams(searchParams);
  };

  const buildLinkParams = (): string => {
    const params = createSearchParams({
      trace: traceId,
      view: viewType,
    } as any);
    (datasources || []).forEach((ds: string) => {
      params.append('datasources', ds);
    });
    return params.toString();
  };

  return (
    <Layout className="trace-view">
      <Header className="linsight-feature-header" style={{ padding: '8px', height: 'unset' }}>
        <div style={{ flex: 1 }}>
          <SpanHeader span={tree[0]} />
        </div>
        {newWindowLink && (
          <Text link={{ target: '_blank', href: `/trace?${buildLinkParams()}` }} style={{ marginRight: 8 }}>
            Open Full Page
            <Icon icon="external-link" style={{ marginLeft: 4 }} />
          </Text>
        )}
        <Dropdown
          render={
            <Dropdown.Menu>
              <Dropdown.Item onClick={() => changeViewType('timeline')} icon={<IconInheritStroked />}>
                Timeline
              </Dropdown.Item>
              <Dropdown.Item onClick={() => changeViewType('flame')} icon={<IconDescend />}>
                Flame Graph
              </Dropdown.Item>
              <Dropdown.Item onClick={() => changeViewType('map')} icon={<IconFlowChartStroked />}>
                Map
              </Dropdown.Item>
              <Dropdown.Item onClick={() => changeViewType('summary')} icon={<IconPieChart2Stroked />}>
                Summary
              </Dropdown.Item>
            </Dropdown.Menu>
          }>
          <Button icon={renderIcon()}>Timeline</Button>
        </Dropdown>
      </Header>
      {renderContent()}
    </Layout>
  );
};

export default TraceView;
