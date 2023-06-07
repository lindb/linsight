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

import { Collapse, Form, Radio } from '@douyinfe/semi-ui';
import { OptionsEditorProps, OrientationType } from '@src/types';
import { ColorMode, JustifyMode, TextMode } from './types';

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
              onOptionsChange({ options: values });
            }
          }}>
          <Form.RadioGroup field="orientation" label="Orientation" type="button" extraText="Layout orientation">
            <Radio value={OrientationType.vertical}>Vertical</Radio>
            <Radio value={OrientationType.horizontal}>Horizontal</Radio>
          </Form.RadioGroup>
          <Form.RadioGroup field="colorMode" label="Color mode" type="button">
            <Radio value={ColorMode.none}>None</Radio>
            <Radio value={ColorMode.value}>Value</Radio>
            <Radio value={ColorMode.background}>Background</Radio>
          </Form.RadioGroup>
          <Form.Select
            field="textMode"
            label="Text mode"
            extraText="Control if name and value is displayed or just name"
            optionList={[
              { value: TextMode.name, label: 'Name', showTick: false },
              { value: TextMode.value, label: 'Value', showTick: false },
              { value: TextMode.valueAndName, label: 'Name and value', showTick: false },
              { value: TextMode.none, label: 'None', showTick: false },
            ]}
          />
          <Form.RadioGroup field="justifyMode" label="Text alignment" type="button">
            <Radio value={JustifyMode.auto}>Auto</Radio>
            <Radio value={JustifyMode.center}>Center</Radio>
          </Form.RadioGroup>
        </Form>
      </Collapse.Panel>
    </Collapse>
  );
};
