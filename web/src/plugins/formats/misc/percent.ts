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

import { FormatCate, Formatted, Formatter, toFixed } from '@src/types';

const transformPercent = (input: number | null, decimals: number = 2): Formatted => {
  if (!input) {
    return { value: '' };
  }
  const value = toFixed(input, decimals);
  return { value: value, suffix: '%' };
};

class Percent extends Formatter {
  constructor() {
    super({
      category: FormatCate.Misc,
      label: 'Percent (0~100)',
      value: 'percent',
    });
  }
  format(input: number | null, decimals?: number | undefined): Formatted {
    return transformPercent(input, decimals);
  }
}

class PercentUnit extends Formatter {
  constructor() {
    super({
      category: FormatCate.Misc,
      label: 'Percent (0~1.0)',
      value: 'percentunit',
    });
  }
  format(input: number | null, decimals?: number | undefined): Formatted {
    if (!input) {
      return { value: '' };
    }
    return transformPercent(input * 100, decimals);
  }
}

export { Percent, PercentUnit };
