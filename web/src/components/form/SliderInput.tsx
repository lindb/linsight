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
import { InputGroup, InputNumber, Slider } from '@douyinfe/semi-ui';
import React from 'react';

const SliderInput: React.FC<{
  label: string;
  initValue?: number;
  min: number;
  max: number;
  step: number;
  marks: {
    [key: number]: string;
  };
  onChange: (val: number) => void;
}> = (props) => {
  const { label, initValue, min, max, step, marks, onChange } = props;
  const onNumberChange = (val: number) => {
    if (val >= min && val <= max) {
      onChange(val);
    }
  };
  return (
    <InputGroup className="linsight-slider-input-group" label={{ text: label }}>
      <Slider
        value={initValue}
        min={min}
        max={max}
        step={step}
        marks={marks}
        onChange={(val: any) => {
          if (val !== initValue) {
            onNumberChange(val);
          }
        }}
      />
      <InputNumber value={initValue} style={{ width: 70 }} min={min} max={max} onNumberChange={onNumberChange} />
    </InputGroup>
  );
};

export default SliderInput;
