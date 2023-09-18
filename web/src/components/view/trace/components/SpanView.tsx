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
import { SideSheet, Tag, Tabs, TabPane, Descriptions } from '@douyinfe/semi-ui';
import React, { useEffect, useRef, useState } from 'react';
import moment from 'moment';
import { AnnotationType, Span } from '@src/types';
import { get } from 'lodash-es';
import { TraceKit } from '@src/utils';
import SpanHeader from './SpanHeader';
import { Resizable } from 're-resizable';
import { TraceViewStore, AnnotationStore } from '@src/stores';
import { observer } from 'mobx-react-lite';
import { useSearchParams } from 'react-router-dom';
import { DashboardView } from '@src/components';
import { DateTimeFormat } from '@src/constants';
import EventView from './EventView';

const Dashboard: React.FC<{ span: Span; resizeStop: boolean }> = (props) => {
  const { span, resizeStop } = props;
  const dashboardId = '77NWe4jVz';
  const timeRange = TraceKit.calcMetricQueryTimeRange(span);

  useEffect(() => {
    AnnotationStore.setAnnotations([
      {
        type: AnnotationType.Span,
        timestamp: Math.floor(span.startTime / 1000000),
        data: {
          span: span,
        },
      },
    ]);
  }, [span]);

  useEffect(() => {
    // NOTE: hack: tiger window resize event, then resize dashboard view
    window.dispatchEvent(new Event('resize'));
  }, [resizeStop]);

  return (
    <DashboardView
      dashboardId={dashboardId}
      initVariableValues={{
        from: moment(timeRange.from).format(DateTimeFormat),
        to: moment(timeRange.to).format(DateTimeFormat),
      }}
    />
  );
};

const SpanView: React.FC<{ span?: Span }> = (props) => {
  const { span } = props;
  const initWidth = useRef(document.body.clientWidth * 0.65);
  const [width, setWidth] = useState(initWidth.current);
  const [finishResize, setFinishResize] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();

  if (!span) {
    return null;
  }
  const events = get(span, 'events', []);

  return (
    <SideSheet
      className="split-view"
      closeOnEsc
      closable={true}
      title={<SpanHeader span={span} />}
      size="large"
      width={width}
      height={'100vh'}
      motion={false}
      mask={false}
      visible={TraceViewStore.viewVisibleSpanView}
      onCancel={() => {
        TraceViewStore.setVisibleSpanView(false);
      }}>
      <Resizable
        defaultSize={{ width: width, height: '100%' }}
        handleClasses={{ left: 'left-handler' }}
        enable={{ left: true }}
        onResizeStop={(_e, _direction, _ref, d) => {
          initWidth.current += d.width;
          setFinishResize(!finishResize);
        }}
        onResize={(_e, _direction, _ref, d) => {
          setWidth(d.width + initWidth.current);
        }}>
        <Tabs
          type="line"
          size="small"
          tabPaneMotion={false}
          activeKey={searchParams.get('tab') || 'tags'}
          onChange={(active: string) => {
            searchParams.set('tab', active);
            setSearchParams(searchParams);
          }}>
          <TabPane tab="Process" itemKey="process">
            <Descriptions data={TraceKit.toKeyValueList(get(span, 'process.tags', {}))} />
          </TabPane>
          <TabPane tab="Tags" itemKey="tags">
            <Descriptions data={TraceKit.toKeyValueList(get(span, 'tags', {}))} />
          </TabPane>
          <TabPane tab="Runtime" itemKey="service">
            <Dashboard span={span} resizeStop={finishResize} />
          </TabPane>
          <TabPane
            tab={
              <>
                Events <Tag>{events.length}</Tag>
              </>
            }
            itemKey="events">
            <EventView span={span} />
          </TabPane>
        </Tabs>
      </Resizable>
    </SideSheet>
  );
};

export default observer(SpanView);
