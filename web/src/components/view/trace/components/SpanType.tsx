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
import { Tag } from '@douyinfe/semi-ui';
import IntegrationIcon from '@src/components/common/IntegrationIcon';
import { IconRepositoryInst, Span } from '@src/types';
import { ColorKit, StringKit, TraceKit } from '@src/utils';
import { isEmpty } from 'lodash-es';
import React from 'react';

const SpanType: React.FC<{ span: Span }> = (props) => {
  const { span } = props;
  const spanType = TraceKit.getSpanType(span);
  if (isEmpty(spanType)) {
    return null;
  }
  const iconCls = IconRepositoryInst.getIconCls(spanType);
  if (isEmpty(iconCls)) {
    return (
      <Tag
        type="solid"
        size="small"
        style={{ padding: '2px', backgroundColor: ColorKit.getColor(StringKit.hashcode(spanType)) }}>
        {spanType}
      </Tag>
    );
  }
  return <IntegrationIcon integration={spanType || 'Unknown'} />;
};

export default SpanType;
