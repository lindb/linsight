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
import { Card } from '@douyinfe/semi-ui';
import { FlamegraphRenderer } from '@pyroscope/flamegraph';
import '@pyroscope/flamegraph/dist/index.css';
import { Trace } from '@src/types';
import React from 'react';
import { TraceKit } from '@src/utils';

const FlameView: React.FC<{ traces: Trace[] }> = (props) => {
  const { traces } = props;
  return (
    <Card className="linsight-feature" bodyStyle={{ padding: 8 }}>
      <FlamegraphRenderer
        profile={TraceKit.convertTraceToProfile(traces)}
        // onlyDisplay="both"
        // onlyDisplay="sandwich"
        onlyDisplay="flamegraph"
        showToolbar={false}
        showCredit={false}
      />
    </Card>
  );
};

export default FlameView;
