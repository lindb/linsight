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
import React, { useContext } from 'react';
import { DatasourceInstance } from '@src/types';
import { Card } from '@douyinfe/semi-ui';
import { TraceView, QueryEditor } from '@src/components';
import Panel from '../dashboard/Panel';
import { PanelEditContext } from '@src/contexts';
import { get } from 'lodash-es';
import { DatasourceKit } from '@src/utils';

const DataExplore: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const { panel } = useContext(PanelEditContext);
  const renderContent = () => {
    if (DatasourceKit.isTrace(datasource)) {
      return (
        <TraceView
          newWindowLink
          traceId={get(panel, 'targets[0].request.traceId')}
          datasources={[datasource.setting.uid]}
        />
      );
    }
    return <Panel panel={panel} />;
  };
  return (
    <>
      <Card className="linsight-feature" bodyStyle={{ padding: 8 }}>
        <QueryEditor datasource={datasource} />
      </Card>
      <div className="linsight-feature" style={{ marginTop: 0, height: 300 }}>
        {renderContent()}
      </div>
    </>
  );
};

export default DataExplore;
