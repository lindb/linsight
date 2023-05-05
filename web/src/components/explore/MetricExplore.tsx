import React from 'react';
import { DatasourceStore } from '@src/stores';
import { DatasourceInstance, DatasourceRepositoryInst } from '@src/types';
import * as _ from 'lodash-es';
import { Card, Collapse, Divider, Typography } from '@douyinfe/semi-ui';
import { IconPlus, IconCopy } from '@douyinfe/semi-icons';
import Panel from '../dashboard/Panel';
const { Text } = Typography;

const MetricExplore: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;
  const QueryEditor = plugin.components.QueryEditor;
  if (!QueryEditor) {
    return null;
  }
  return (
    <>
      <Card className="linsight-feature">
        <div className="linsight-explore">
          <Collapse activeKey={['A']} expandIconPosition="left">
            <Collapse.Panel
              header={
                <div style={{ display: 'flex', width: '100%', alignItems: 'center' }}>
                  <div style={{ flex: 1 }}>
                    <Text>A</Text>
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <IconCopy />
                  </div>
                </div>
              }
              itemKey="A">
              <QueryEditor
                datasource={datasource}
                onChange={(values: object) => console.log('query editor...', values)}
              />
            </Collapse.Panel>
          </Collapse>
        </div>
      </Card>
      <div className="linsight-feature" style={{ marginTop: 0, height: 300 }}>
        <Panel
          panel={{
            type: 'timeseries',
            targets: [{ datasrouce: { uid: datasource.setting.uid }, request: { sql: 'sss' } }],
          }}
        />
      </div>
    </>
  );
};

export default MetricExplore;
