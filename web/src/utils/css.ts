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
import { ThemeType } from '@src/types';

const setStyle = (target: any, style: any): void => {
  if (!target || !target.style) {
    return;
  }
  Object.keys(style).forEach((key) => {
    target.style[key] = style[key];
  });
};

/**
 * Get color value from body style variables
 */
const getColor = (key: string, theme: ThemeType): string => {
  const body = document.body;
  body.setAttribute('theme-mode', theme);
  const bodyStyles = getComputedStyle(body);
  return bodyStyles.getPropertyValue(key).trim();
};

export default {
  getColor,
  setStyle,
};
