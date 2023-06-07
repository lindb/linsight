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
import { FormatCate, Formatted, Formatter } from '@src/types';
import { FormatKit } from '@src/utils';

class Short extends Formatter {
  private units: string[];
  constructor() {
    super({
      category: FormatCate.Misc,
      label: 'Short',
      value: 'short',
    });
    this.units = ['', ' K', ' Mil', ' Bil', ' Tri', ' Quadr', ' Quint', ' Sext', ' Sept'];
  }
  format(input: number | null, decimals?: number | undefined): Formatted {
    return FormatKit.formatUnit(input, this.units, 1000, decimals);
  }
}

export { Short };
