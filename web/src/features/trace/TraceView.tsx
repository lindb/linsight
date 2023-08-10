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
import { Button, Dropdown, Layout } from '@douyinfe/semi-ui';
import { IconInheritStroked, IconDescend, IconFlowChartStroked, IconPieChart2Stroked } from '@douyinfe/semi-icons';
import { useRequest, useTrace } from '@src/hooks';
import { DataQuerySrv } from '@src/services';
import React, { useState } from 'react';
import TimelineView from './timeline/TimelineView';
import { Trace } from '@src/types';
import FlameView from './flame/FlameView';
import { cloneDeep } from 'lodash-es';
import { Loading } from '@src/components';
import './trace-view.scss';
import { useSearchParams } from 'react-router-dom';

const { Header } = Layout;

const TraceView: React.FC = () => {
  const [viewType, setViewType] = useState('timeline');
  const [searchParams] = useSearchParams();
  const { loading, result } = useTrace({
    datasource: { uid: 'dqOPrJ6Vk' },
    request: {
      traceId: `${searchParams.get('traceId')}`,
    },
  });
  if (loading) {
    return (
      <div className="loading">
        <Loading />
      </div>
    );
  }
  console.error(result);
  if (!result) {
    return null;
  }
  const renderContent = () => {
    switch (viewType) {
      case 'flame':
        return <FlameView traces={cloneDeep(result as Trace[])} />;
      default:
        return <TimelineView traces={cloneDeep(result as Trace[])} />;
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
  return (
    <Layout className="trace-view">
      <Header className="linsight-feature-header" style={{ padding: '0 12px' }}>
        <div style={{ flex: 1 }}>adddd</div>
        <Dropdown
          render={
            <Dropdown.Menu>
              <Dropdown.Item onClick={() => setViewType('timeline')} icon={<IconInheritStroked />}>
                Timeline
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setViewType('flame')} icon={<IconDescend />}>
                Flame Graph
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setViewType('map')} icon={<IconFlowChartStroked />}>
                Map
              </Dropdown.Item>
              <Dropdown.Item onClick={() => setViewType('summary')} icon={<IconPieChart2Stroked />}>
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
