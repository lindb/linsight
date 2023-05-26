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
import { OptionsEditorProps } from '@src/types';
import React from 'react';
import { get, cloneDeep, has, set } from 'lodash-es';
import './components/timeseries.scss';
import { SliderInput } from '@src/components';
import Smooth from '@src/plugins/visualizations/timeseries/images/smooth.svg';
import Line from '@src/plugins/visualizations/timeseries/images/line.svg';
import ActiveSmooth from '@src/plugins/visualizations/timeseries/images/active-smooth.svg';
import ActiveLine from '@src/plugins/visualizations/timeseries/images/active-line.svg';
import { LegendMode, Placement } from './types';

const LegendForm: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;
  return (
    <Form
      className="linsight-form linsight-panel-setting"
      layout="vertical"
      initValues={get(panel, 'options.legend', {})}
      onValueChange={(values: object) => {
        if (onOptionsChange) {
          onOptionsChange({ legend: values });
        }
      }}>
      <Form.RadioGroup field="mode" label="Mode" type="button">
        <Radio value={LegendMode.List}>List</Radio>
        <Radio value={LegendMode.Table}>Table</Radio>
        <Radio value={LegendMode.Hidden}>Hidden</Radio>
      </Form.RadioGroup>
      <Form.RadioGroup field="placement" label="Placement" type="button">
        <Radio value={Placement.Bottom}>Bottom</Radio>
        <Radio value={Placement.Right}>Right</Radio>
      </Form.RadioGroup>
      <Form.Select
        field="values"
        label="Values"
        placeholder="choose"
        style={{ width: '100%' }}
        showClear
        multiple
        optionList={[
          { value: 'total', label: 'Total' },
          { value: 'count', label: 'Count' },
          { value: 'mean', label: 'Mean' },
          { value: 'min', label: 'Min' },
          { value: 'max', label: 'Max' },
          { value: 'first', label: 'First' },
          { value: 'last', label: 'Last' },
        ]}
      />
    </Form>
  );
};

const GraphForm: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;

  const isLineChart = (values: any): boolean => {
    return get(values, 'type', 'line') === 'line';
  };

  const showPointSize = (values: any): boolean => {
    return isLineChart(values) && get(values, 'points', 'always') === 'always';
  };

  return (
    <Form
      className="linsight-form linsight-panel-setting"
      layout="vertical"
      initValues={panel.options}
      onValueChange={(values: object) => {
        if (onOptionsChange) {
          const options = cloneDeep(values);
          if (!has(options, 'max')) {
            set(options, 'max', Number.NaN);
          }
          if (!has(options, 'min')) {
            set(options, 'min', Number.NaN);
          }
          onOptionsChange(options);
        }
      }}
      render={({ formApi, values }: any) => {
        const lineInterpolation = get(values, 'lineInterpolation', 'line');
        return (
          <>
            <Form.RadioGroup
              field="type"
              label="Mode"
              type="button"
              onChange={(e) => {
                if (e.target.value == 'point') {
                  formApi.setValue('lineWidth', 0);
                }
              }}>
              <Radio value="line">Line</Radio>
              <Radio value="bar">Bar</Radio>
            </Form.RadioGroup>
            {/*use display fix miss radio value*/}
            <div style={{ display: isLineChart(values) ? 'block' : 'none' }}>
              <Form.RadioGroup
                initValue={get(values, 'lineInterpolation')}
                field="lineInterpolation"
                label="Line interpolation"
                type="button">
                <Radio value="line">
                  <img src={`${lineInterpolation === 'line' ? ActiveLine : Line}`} />
                </Radio>
                <Radio value="monotone">
                  <img src={`${lineInterpolation === 'monotone' ? ActiveSmooth : Smooth}`} />
                </Radio>
              </Form.RadioGroup>
              <Form.Slot style={{ paddingBottom: 8 }}>
                <SliderInput
                  initValue={get(values, 'lineWidth', 1)}
                  label="Line width"
                  min={0}
                  max={10}
                  step={1}
                  marks={{ 0: '0', 10: '10' }}
                  onChange={(val: number) => formApi.setValue('lineWidth', val)}
                />
              </Form.Slot>
              <Form.Slot style={{ paddingBottom: 8 }}>
                <SliderInput
                  initValue={get(values, 'fillOpacity', 0)}
                  label="Fill opacity"
                  min={0}
                  max={100}
                  step={5}
                  marks={{ 0: '0', 100: '100' }}
                  onChange={(val: number) => formApi.setValue('fillOpacity', val)}
                />
              </Form.Slot>
              <Form.RadioGroup initValue={get(values, 'lineStyle')} field="lineStyle" label="Line style" type="button">
                <Radio value="solid">Solid</Radio>
                <Radio value="dash">Dash</Radio>
              </Form.RadioGroup>
              <Form.RadioGroup field="spanNulls" label="Connect null values" type="button">
                <Radio value="false">Never</Radio>
                <Radio value="true">Always</Radio>
              </Form.RadioGroup>
              <Form.RadioGroup
                field="points"
                label="Show points"
                type="button"
                onChange={(e) => {
                  if (e.target.value == 'never') {
                    formApi.setValue('pointSize', 0);
                  } else {
                    formApi.setValue('pointSize', 1);
                  }
                }}>
                <Radio value="never">Never</Radio>
                <Radio value="always">Always</Radio>
              </Form.RadioGroup>
            </div>
            <Form.Slot style={{ paddingBottom: 8, display: showPointSize(values) ? 'block' : 'none' }}>
              <SliderInput
                initValue={get(values, 'pointSize', 2)}
                label="Point size"
                min={0}
                max={10}
                step={1}
                marks={{ 0: '0', 10: '10' }}
                onChange={(val: number) => formApi.setValue('pointSize', val)}
              />
            </Form.Slot>
            <Form.InputNumber field="min" label="Min" />
            <Form.InputNumber field="max" label="Max" />
          </>
        );
      }}
    />
  );
};

export const OptionsEditor: React.FC<OptionsEditorProps> = (props) => {
  return (
    <Collapse expandIconPosition="left" defaultActiveKey={['graph', 'legend']}>
      <Collapse.Panel header="Graph syltes" itemKey="graph">
        <GraphForm {...props} />
      </Collapse.Panel>
      <Collapse.Panel header="Legend" itemKey="legend">
        <LegendForm {...props} />
      </Collapse.Panel>
    </Collapse>
  );
};
