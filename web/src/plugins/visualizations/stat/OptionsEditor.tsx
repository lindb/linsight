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
import React from 'react';

import { Collapse, Form } from '@douyinfe/semi-ui';
import { OptionsEditorProps } from '@src/types';

/**
 * Text options editor
 *
 * options type: @typedef {GaugeOptions}
 */
export const OptionsEditor: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;
  return (
    <Collapse expandIconPosition="left" defaultActiveKey="stat">
      <Collapse.Panel header="Stat styles" itemKey="stat">
        <Form
          extraTextPosition="middle"
          className="linsight-form linsight-panel-setting"
          initValues={panel.options}
          onValueChange={(values: object) => {
            if (onOptionsChange) {
              onOptionsChange(values);
            }
          }}>
          <Form.Switch
            field="showThresholdMarkers"
            label="Show threshold markers"
            extraText="Redners the thresholds as an outer bar"
          />
        </Form>
      </Collapse.Panel>
    </Collapse>
  );
};
