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
import { Card, Space, Typography } from '@douyinfe/semi-ui';
import { IntegrationIcon } from '@src/components';
import { FormatRepositoryInst, Span } from '@src/types';
import moment from 'moment';
import React from 'react';
import SpanStatus from './SpanStatus';

const { Text } = Typography;

const SpanHeader: React.FC<{ span: Span }> = (props) => {
  const { span } = props;
  const language = span.process.sdkLanguage;
  const serviceName = span.process.serviceName;
  return (
    <Card.Meta
      title={
        <Space>
          <IntegrationIcon integration={language} />
          <Text strong>{serviceName}</Text>
          <Text
            strong
            ellipsis={{ showTooltip: true }}
            style={{ marginLeft: 4, marginTop: 2, width: 500 }}
            type="tertiary">
            {span.name}
          </Text>
        </Space>
      }
      description={
        <Space>
          <Text strong size="small">
            Kind:
          </Text>
          <Text type="tertiary" size="small">
            {span.kind}
          </Text>
          <Text strong size="small">
            Start Time:
          </Text>
          <Text type="tertiary" size="small">
            {moment(span.startTime / 1000000).format('YYYY-MM-DD HH:mm:ss.SSS')}
          </Text>
          <Text strong size="small">
            Duration:
          </Text>
          <Text type="tertiary" size="small">
            {FormatRepositoryInst.get('ns').formatString(span.duration, 3)}
          </Text>
          <Text strong size="small">
            Span ID:
          </Text>
          <Text type="tertiary" size="small">
            {span.spanId}
          </Text>
          <SpanStatus span={span} />
        </Space>
      }
    />
  );
};

export default SpanHeader;
