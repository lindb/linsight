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
import { toLower, sortBy, join } from 'lodash-es';
export enum FormatCate {
  Misc = 'Misc',
  Data = 'Data',
  DataRate = 'Data Rate',
  Time = 'Time',
  Throughtput = 'Throughtput',
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

const cateToString = (cate: FormatCate): string => {
  return toLower(cate).replaceAll(' ', '');
};

const categoryToString = (category: FormatCategory): string => {
  return [cateToString(category.category), category.value].join('_');
};

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

  public register(formatter: Formatter): FormatRepository {
    this.formatters.set(categoryToString(formatter.Category), formatter);
    return this;
  }

  public get(category: any): Formatter {
    const format = this.formatters.get(join(category, '_'));
    if (format) {
      return format;
    }
    return defaultFormatter;
  }

  public formatString(category: any, input: number | null, decimals = 2): string {
    return this.get(category).formatString(input, decimals);
  }

  public tree(): any {
    const keys = sortBy(Array.from(this.formatters.keys()));
    const result: any[] = [];
    let cate: FormatCate;
    let children: any[];
    keys.forEach((key: string) => {
      const formatter = this.formatters.get(key);
      if (formatter) {
        if (formatter.Category.category !== cate) {
          cate = formatter.Category.category;
          children = [];
          const cateStr = cateToString(cate);
          result.push({
            label: cate,
            value: cateStr,
            key: cateStr,
            children: children,
          });
        }
        children.push({
          key: formatter.Category.label,
          label: formatter.Category.label,
          value: formatter.Category.value,
        });
      }
    });
    return result;
  }
}

export const FormatRepositoryInst = new FormatRepository();

FormatRepositoryInst.register(new DefaultFormatter());
export { toFixed, formattedToString };
