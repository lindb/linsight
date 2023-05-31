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
import { Legend, LegendDisplayMode, LegendPlacement, OptionsEditorProps, ThemeType } from '@src/types';
import React, { useContext } from 'react';
import { get, cloneDeep, has, set } from 'lodash-es';
import './components/timeseries.scss';
import { SliderInput } from '@src/components';
import Smooth from '@src/plugins/visualizations/timeseries/images/smooth.svg';
import Step from '@src/plugins/visualizations/timeseries/images/step.svg';
import ActiveStep from '@src/plugins/visualizations/timeseries/images/active-step.svg';
import DarkStep from '@src/plugins/visualizations/timeseries/images/step-dark.svg';
import Line from '@src/plugins/visualizations/timeseries/images/line.svg';
import DarkSmooth from '@src/plugins/visualizations/timeseries/images/smooth-dark.svg';
import DarkLine from '@src/plugins/visualizations/timeseries/images/line-dark.svg';
import ActiveSmooth from '@src/plugins/visualizations/timeseries/images/active-smooth.svg';
import ActiveLine from '@src/plugins/visualizations/timeseries/images/active-line.svg';
import { getCustomOptions, TimeSeriesOptions } from './types';
import { PlatformContext } from '@src/contexts';

const LegendForm: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;
  return (
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
              field="calcs"
              label="Values"
              placeholder="choose"
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
          </div>
        </>
      )}></Form>
  );
};

const GraphForm: React.FC<OptionsEditorProps> = (props) => {
  const { panel, onOptionsChange } = props;
  const { theme } = useContext(PlatformContext);

  const isLineChart = (values: any): boolean => {
    return get(values, 'drawStyle', 'line') === 'line';
  };

  const showPointSize = (values: TimeSeriesOptions): boolean => {
    return values.drawStyle !== 'bars' && get(values, 'showPoints', 'always') === 'always';
  };

  return (
    <Form
      className="linsight-form linsight-panel-setting"
      layout="vertical"
      initValues={getCustomOptions(panel)}
      onValueChange={(values: TimeSeriesOptions) => {
        if (onOptionsChange) {
          const options = cloneDeep(values);
          if (!has(options, 'max')) {
            set(options, 'max', Number.NaN);
          }
          if (!has(options, 'min')) {
            set(options, 'min', Number.NaN);
          }
          onOptionsChange({
            fieldConfig: {
              defaults: {
                custom: values,
              },
            },
          });
        }
      }}
      render={({ formApi, values }: any) => {
        const lineInterpolation = get(values, 'lineInterpolation', 'line');
        return (
          <>
            <Form.RadioGroup field="drawStyle" label="Style" type="button">
              <Radio value="line">Lines</Radio>
              <Radio value="bars">Bars</Radio>
              <Radio value="points">Points</Radio>
            </Form.RadioGroup>
            {/*use display fix miss radio value*/}
            <div style={{ display: isLineChart(values) ? 'block' : 'none' }}>
              <Form.RadioGroup
                // initValue={get(values, 'lineInterpolation')}
                field="lineInterpolation"
                label="Line interpolation"
                type="button">
                <Form.Radio value="linear">
                  <img
                    src={`${lineInterpolation === 'linear' ? ActiveLine : theme === ThemeType.Dark ? DarkLine : Line}`}
                  />
                </Form.Radio>
                <Form.Radio value="smooth">
                  <img
                    src={`${
                      lineInterpolation === 'smooth' ? ActiveSmooth : theme === ThemeType.Dark ? DarkSmooth : Smooth
                    }`}
                  />
                </Form.Radio>
                <Form.Radio value="step">
                  <img
                    src={`${
                      ['step', 'stepBefore', 'stepAfter'].indexOf(lineInterpolation) >= 0
                        ? ActiveStep
                        : theme === ThemeType.Dark
                        ? DarkStep
                        : Step
                    }`}
                  />
                </Form.Radio>
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
              <Form.RadioGroup field="lineStyle.fill" label="Line style" type="button">
                <Form.Radio value="solid">Solid</Form.Radio>
                <Form.Radio value="dash">Dash</Form.Radio>
                <Form.Radio value="dots">Dots</Form.Radio>
              </Form.RadioGroup>
              <Form.RadioGroup field="spanNulls" label="Connect null values" type="button">
                <Form.Radio value={false as any}>Never</Form.Radio>
                <Form.Radio value={true as any}>Always</Form.Radio>
              </Form.RadioGroup>
              <Form.RadioGroup field="showPoints" label="Show points" type="button">
                <Form.Radio value="never">Never</Form.Radio>
                <Form.Radio value="always">Always</Form.Radio>
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
            <Form.RadioGroup field="axisGridShow" label="Show grid lines" type="button">
              <Form.Radio value={true as any}>On</Form.Radio>
              <Form.Radio value={false as any}>Off</Form.Radio>
            </Form.RadioGroup>
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
