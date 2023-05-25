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
import { DatasourceCategory, Plugin, Query, ThemeType } from '@src/types';
import { cloneDeep } from 'lodash-es';

export interface PanelSetting {
  id?: number;
  title?: string;
  description?: string;
  type?: string;
  options?: object;
  targets?: Query[];
  grid?: {
    x: number;
    y: number;
    w: number;
    h: number;
    i?: string;
  };
}

export interface OptionsEditorProps {
  panel: PanelSetting;
  onOptionsChange?: (options: object) => void;
}

export interface VisualizationProps {
  panel: PanelSetting;
  theme: ThemeType;
  datasets?: any;
}

export interface VisualizationPluginComponents {
  OptionsEditor?: ComponentType<OptionsEditorProps>;
  DefaultOptions?: object;
}

class VisualizationPlugin extends Plugin {
  components: VisualizationPluginComponents = {};
  constructor(
    public category: DatasourceCategory,
    name: string,
    type: string,
    description: string,
    public Visualization: ComponentType<VisualizationProps>
  ) {
    super(name, type, description);
  }

  setOptionsEditor(optionsEditor: ComponentType<any>): VisualizationPlugin {
    this.components.OptionsEditor = optionsEditor;
    return this;
  }
  setDefaultOptions(options: object): VisualizationPlugin {
    this.components.DefaultOptions = options;
    return this;
  }

  getDefaultOptions(): object {
    return cloneDeep(this.components.DefaultOptions) || {};
  }
}

class VisualizationRepository {
  private visualizations: Map<string, VisualizationPlugin> = new Map<string, VisualizationPlugin>();

  public register(ds: VisualizationPlugin): VisualizationRepository {
    this.visualizations.set(ds.Type, ds);
    return this;
  }

  public getPlugins(): VisualizationPlugin[] {
    const rs: VisualizationPlugin[] = [];
    for (const visualization of this.visualizations.values()) {
      rs.push(visualization);
    }
    return rs;
  }

  public get(type: string): VisualizationPlugin | undefined {
    return this.visualizations.get(type);
  }
}

export const VisualizationRepositoryInst = new VisualizationRepository();
export { VisualizationPlugin };
