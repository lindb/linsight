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
import { find } from 'lodash-es';

export enum FormatCate {
  Misc = 'Misc',
  Data = 'Data',
  DataRate = 'Data Rate',
  Time = 'Time',
  Throughput = 'Throughput',
}

export interface Formatted {
  prefix?: string;
  value: string;
  suffix?: string;
}

export interface FormatCategory {
  category: FormatCate;
  value: string;
  label: string;
}

const formattedToString = (formatted: Formatted): string => {
  return `${formatted.prefix ?? ''}${formatted.value}${formatted.suffix ?? ''}`;
};

export abstract class Formatter {
  constructor(public Category: FormatCategory) {}
  abstract format(input: number | null, decimals?: number): Formatted;

  formatString(input: number | null, decimals?: number): string {
    return formattedToString(this.format(input, decimals));
  }
}

const toFixed = (value: number | null, decimals: number = 2): string => {
  if (value == null) {
    return '';
  }
  return value.toFixed(decimals);
};

class DefaultFormatter extends Formatter {
  constructor() {
    super({
      category: FormatCate.Misc,
      label: 'None',
      value: 'none',
    });
  }
  format(input: number | null, decimals?: number | undefined): Formatted {
    const value = toFixed(input, decimals);
    return { value };
  }
}

const defaultFormatter = new DefaultFormatter();

class FormatRepository {
  private formatters: Map<string, Formatter> = new Map<string, Formatter>();
  private treeData: any[] = [];

  public register(formatter: Formatter): FormatRepository {
    this.formatters.set(formatter.Category.value, formatter);
    return this;
  }

  public get(unit: string): Formatter {
    const format = this.formatters.get(unit);
    if (format) {
      return format;
    }
    return defaultFormatter;
  }

  public formatString(unit: any, input: number | null, decimals = 2): string {
    return this.get(unit).formatString(input, decimals);
  }

  public tree(): any {
    return this.treeData;
  }

  public buildTree() {
    const result: any[] = [];
    this.formatters.forEach((formatter: Formatter) => {
      const cate = formatter.Category.category;
      const category = find(result, { value: cate });
      const unit = {
        key: formatter.Category.value,
        label: formatter.Category.label,
        value: formatter.Category.value,
      };

      if (category) {
        category.children.push(unit);
      } else {
        result.push({
          label: cate,
          value: cate,
          key: cate,
          children: [unit],
        });
      }
    });
    this.treeData = result;
  }
}

export const FormatRepositoryInst = new FormatRepository();

FormatRepositoryInst.register(new DefaultFormatter());
export { toFixed, formattedToString };
