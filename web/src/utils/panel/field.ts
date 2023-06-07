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
import { DefaultThresholds } from '@src/constants';
import { FieldConfig, Threshold, ThresholdMode } from '@src/types';
import { isEmpty, orderBy, isNil, sortedIndexBy } from 'lodash-es';

/**
 * Return active threshold
 */
const getActiveThreshold = (value: number, thresholds: Threshold[]): Threshold => {
  const idx = sortedIndexBy(thresholds, { value: value }, (o: Threshold) => {
    return o.value;
  });
  if (idx >= (thresholds || []).length) {
    return thresholds[idx - 1];
  }
  return thresholds[idx];
};

/**
 * Return sort and format thresholds
 */
const getFormattedThresholds = (field: FieldConfig): Threshold[] => {
  const thresholds = field.thresholds ?? DefaultThresholds;
  const isPercent = thresholds.mode === ThresholdMode.Percenttag;
  const min = isPercent ? 0 : field.min ?? 0;
  const max = isPercent ? 100 : field.max ?? 100;
  const steps = orderBy(thresholds.steps || [], [(obj) => isNil(obj.value), 'value'], ['desc', 'asc']);
  if (!steps || isEmpty(steps)) {
    return [];
  }
  const first = steps[0];
  if (steps?.length === 1) {
    return [{ value: steps[0].value, _percent: max - min, color: steps[0].color }];
  }
  const last = steps[steps.length - 1];

  const formatted: Threshold[] = [
    {
      value: steps[1].value,
      _percent: steps[1].value,
      color: first.color,
    },
  ];
  for (let i = 1; i < steps.length - 1; i++) {
    const step = steps[i];
    formatted.push({ color: step.color, value: steps[i + 1].value, _percent: steps[i + 1].value - step.value });
  }
  formatted.push({ color: last?.color, _percent: max - last?.value, value: max });
  return formatted;
};

export default {
  getActiveThreshold,
  getFormattedThresholds,
};
