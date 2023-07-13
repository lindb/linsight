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

export enum OrientationType {
  horizontal = 'horizontal',
  vertical = 'vertical',
}

export enum DataSetType {
  TimeSeries = 'timeseries',
  SingleStat = 'singleStat',
}

export interface GridPos {
  x: number;
  y: number;
  w: number;
  h: number;
  i?: string;
}

export interface Legend {
  showLegend?: boolean;
  displayMode?: LegendDisplayMode;
  placement?: LegendPlacement;
  calcs?: string[]; // time series chart
  values?: string[]; // pie chart
}

export enum LegendDisplayMode {
  List = 'list',
  Table = 'table',
}

export enum LegendPlacement {
  Bottom = 'bottom',
  Right = 'right',
}

export interface PanelSetting {
  id?: number;
  datasource?: { uid: string; type?: string };
  title?: string;
  integration?: string;
  description?: string;
  type?: string;
  collapsed?: boolean;
  options?: object;
  targets?: Query[];
  fieldConfig?: Record<string, FieldConfig>;
  panels?: PanelSetting[];
  gridPos?: GridPos;
  libraryPanel?: { name?: string; uid?: string };
}

export interface FieldConfig<VOptions = any> {
  thresholds?: Thresholds;
  unit?: string;
  decimals?: number;
  min?: number;
  max?: number;
  custom?: VOptions;
}

export enum ThresholdMode {
  Absolute = 'absolute',
  Percenttag = 'percentage',
}

export interface Thresholds {
  mode?: ThresholdMode;
  steps?: Threshold[];
}

export interface Threshold {
  color?: string;
  value?: number | null;
  _percent?: number | null;
}

export interface OptionsEditorProps {
  panel: PanelSetting;
  onOptionsChange?: (options: PanelSetting) => void;
}

export interface VisualizationProps {
  panel: PanelSetting;
  theme: ThemeType;
  datasets?: any;
}

export interface VisualizationPluginComponents {
  OptionsEditor?: ComponentType<OptionsEditorProps>;
  DefaultOptions?: PanelSetting;
}

class VisualizationPlugin extends Plugin {
  components: VisualizationPluginComponents = {};
  getDataSetTypeFn?: (options: PanelSetting) => DataSetType;

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

  setDefaultOptions(options: PanelSetting): VisualizationPlugin {
    this.components.DefaultOptions = options;
    return this;
  }

  datesetTypeFn(fn: (options: PanelSetting) => DataSetType): VisualizationPlugin {
    this.getDataSetTypeFn = fn;
    return this;
  }

  getDefaultOptions(): object {
    return cloneDeep(this.components.DefaultOptions) || {};
  }

  getDataSetType(options: PanelSetting): DataSetType {
    if (this.getDataSetTypeFn) {
      return this.getDataSetTypeFn(options);
    }
    return DataSetType.TimeSeries;
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

  public getDefaultOptions(type: string): object {
    const plugin = this.get(type);
    if (!plugin) {
      return {};
    }
    return plugin.getDefaultOptions();
  }

  public get(type: string): VisualizationPlugin {
    const plugin = this.visualizations.get(type);
    if (plugin) {
      return plugin;
    }
    throw new Error('Visualization plugin not implemented.');
  }
}

export const VisualizationRepositoryInst = new VisualizationRepository();
export { VisualizationPlugin };
