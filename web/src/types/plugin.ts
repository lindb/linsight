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
import { ComponentType } from 'react';
import { ThemeType } from './theme';

abstract class Plugin {
  darkLogo?: ComponentType;
  lightLogo?: ComponentType;
  constructor(public Name: string, public Type: string, public Description: string) {}

  setDarkLogo(dark: ComponentType): Plugin {
    this.darkLogo = dark;
    return this;
  }

  setLightLogo(light: ComponentType): Plugin {
    this.lightLogo = light;
    return this;
  }

  getLogo(theme: ThemeType): ComponentType | undefined {
    return theme === ThemeType.Dark ? this.darkLogo : this.lightLogo;
  }
}

export { Plugin };
