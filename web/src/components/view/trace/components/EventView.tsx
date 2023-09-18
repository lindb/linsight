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
import { Timeline, Typography } from '@douyinfe/semi-ui';
import { Span, Event } from '@src/types';
import { get } from 'lodash-es';
import moment from 'moment';
import React from 'react';
const { Text } = Typography;

const EventView: React.FC<{ span: Span }> = (props) => {
  const { span } = props;
  const events = get(span, 'events', []);
  return (
    <Timeline
      dataSource={events.map((event: Event, _idx: number) => {
        return {
          content: <Text size="small">{moment(event.timestamp / 1000000).format('YYYY-MM-DD HH:mm:ss.SSS')}</Text>,
          time: event.name,
        };
      })}
    />
  );
};

export default EventView;
