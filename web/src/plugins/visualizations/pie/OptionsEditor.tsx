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
import { Collapse, Form, Radio } from '@douyinfe/semi-ui';
import { Legend, LegendDisplayMode, LegendPlacement, OptionsEditorProps } from '@src/types';
import React from 'react';
import { PieType } from './types';
import { get } from 'lodash-es';

/**
 * Pie options editor
 *
 * options type: @typedef {PieOptions}
 */
export const OptionsEditor: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;
  return (
    <Collapse expandIconPosition="left" defaultActiveKey={['pie', 'legend']}>
      <Collapse.Panel header="Pie chart" itemKey="pie">
        <Form
          extraTextPosition="middle"
          className="linsight-form linsight-panel-setting"
          initValues={panel.options}
          onValueChange={(values: object) => {
            if (onOptionsChange) {
              onOptionsChange({ options: values });
            }
          }}>
          <Form.RadioGroup field="pieType" label="Pie chart type" type="button">
            <Radio value={PieType.pie}>Pie</Radio>
            <Radio value={PieType.donut}>Doughnut</Radio>
          </Form.RadioGroup>
        </Form>
      </Collapse.Panel>
      <Collapse.Panel header="Legend" itemKey="legend">
        <Form
          className="linsight-form linsight-panel-setting"
          layout="vertical"
          initValues={get(panel, 'options.legend', {})}
          onValueChange={(values: Legend) => {
            if (onOptionsChange) {
              onOptionsChange({ options: { legend: values } });
            }
          }}
          render={({ values }: any) => (
            <>
              <Form.Switch field="showLegend" label="Visibility" />
              <div style={{ display: values.showLegend ? 'block' : 'none' }}>
                <Form.RadioGroup field="displayMode" label="Mode" type="button">
                  <Radio value={LegendDisplayMode.List}>List</Radio>
                  <Radio value={LegendDisplayMode.Table}>Table</Radio>
                </Form.RadioGroup>
                <Form.RadioGroup field="placement" label="Placement" type="button">
                  <Radio value={LegendPlacement.Bottom}>Bottom</Radio>
                  <Radio value={LegendPlacement.Right}>Right</Radio>
                </Form.RadioGroup>
                <Form.Select
                  field="values"
                  label="Values"
                  placeholder="choose"
                  showClear
                  multiple
                  optionList={[
                    { value: 'percent', label: 'Percent' },
                    { value: 'value', label: 'Value' },
                  ]}
                />
              </div>
            </>
          )}></Form>
      </Collapse.Panel>
    </Collapse>
  );
};
