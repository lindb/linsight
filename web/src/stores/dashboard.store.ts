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
import { Notification } from '@src/components';
import { DefaultColumns, RowPanelType, VisualizationAddPanelType } from '@src/constants';
import { DashboardSrv } from '@src/services';
import { Dashboard, PanelSetting, Variable } from '@src/types';
import { ApiKit, ObjectKit } from '@src/utils';
import {
  set,
  get,
  find,
  cloneDeep,
  has,
  concat,
  findIndex,
  forIn,
  merge,
  pick,
  isEmpty,
  pullAt,
  maxBy,
} from 'lodash-es';
import { makeAutoObservable, toJS } from 'mobx';

class DashboardStore {
  public dashboard: Dashboard = {};
  private panelSeq = 0;

  constructor() {
    makeAutoObservable(this);
  }

  private assignPanelId(): number {
    this.panelSeq++;
    return this.panelSeq;
  }

  getPanels(): PanelSetting[] {
    return get(this.dashboard, 'config.panels', []);
  }

  getPanel(panelId: number | undefined): PanelSetting | undefined {
    console.log('xxxxxxxx get panel', panelId, toJS(this.dashboard));
    const panels = this.getPanels();
    return find(panels, { id: panelId });
  }

  getPanelsOfRow(rowPanelId: number | undefined): PanelSetting[] {
    const panels = this.getPanels();
    const result: PanelSetting[] = [];
    const index = findIndex(panels, { id: rowPanelId });
    if (index < 0) {
      return result;
    }
    for (let i = index + 1; i < panels.length; i++) {
      const panel = panels[i];
      if (panel.type === RowPanelType) {
        break;
      }
      result.push(panel);
    }
    return result;
  }

  collapseRow(row: PanelSetting, collapsed: boolean) {
    this.updatePanelConfig(row, { collapsed: collapsed });
    const children = this.getPanelsOfRow(row.id);
    (children || []).forEach((child: PanelSetting) => set(child, '_hidden', collapsed));
  }

  clonePanel(panel: PanelSetting) {
    const newPanel: PanelSetting = cloneDeep(panel);
    //FIXME: remove other props?
    newPanel.id = this.assignPanelId();
    if (panel.grid?.x + 2 * panel.grid?.w <= DefaultColumns) {
      newPanel.grid.x += panel.grid.w;
    } else {
      newPanel.grid.y += panel.grid.y;
    }
    this.addPanel(newPanel);
  }

  createPanelConfig(newID: boolean = true): PanelSetting {
    const cfg: PanelSetting = {
      title: 'Add panel',
      type: VisualizationAddPanelType,
      grid: { w: 12, h: 7, x: 0, y: 0 },
    };
    if (newID) {
      cfg.id = this.assignPanelId();
    }
    return cfg;
  }

  addPanel(panel: PanelSetting) {
    this.setPanelGridId(panel);
    const firstPanelType = get(this.dashboard, 'config.panels[0].type', null);
    if (firstPanelType === VisualizationAddPanelType) {
      // first panel is add panel widget, ignore this panel
      return;
    }
    if (has(this.dashboard, 'config.panels')) {
      set(this.dashboard, 'config.panels', concat(panel, this.dashboard.config?.panels));
    } else {
      set(this.dashboard, 'config.panels', [panel]);
    }
    this.sortPanels();
  }

  deleteRowAndChildren(row: PanelSetting) {
    const children = this.getPanelsOfRow(row.id);
    (children || []).forEach((child: PanelSetting) => this.deletePanel(child));
    this.deletePanel(row);
  }

  deletePanel(panel: PanelSetting) {
    const panels = this.getPanels();
    const index = findIndex(panels, { id: panel.id });
    if (index >= 0) {
      pullAt(panels, index);
      this.sortPanels();
    }
  }

  updatePanel(panel: PanelSetting) {
    const panels = this.getPanels();
    const index = findIndex(panels, { id: panel.id });
    if (index >= 0) {
      panels[index] = panel;
      this.sortPanels();
    }
  }

  updatePanelConfig(panel: any, config: any) {
    forIn(config, function (value, key) {
      set(panel, key, value);
    });
    this.setPanelGridId(panel);
  }

  private setPanelGridId(panel: any) {
    // NOTE: make sure set grid i
    set(panel, 'grid.i', `${panel.id}`);
  }

  updateDashboardProps(values: any) {
    console.log(toJS(this.dashboard));
    this.dashboard = merge(this.dashboard, values);
  }

  getVariables(): Variable[] {
    return get(this.dashboard, 'config.variables', []);
  }

  addVariable(variable: any) {
    if (has(this.dashboard, 'config.variables')) {
      set(this.dashboard, 'config.variables', concat(this.dashboard.config?.variables, variable));
    } else {
      set(this.dashboard, 'config.variables', [variable]);
    }
  }

  updateVariable(index: string, variable: any) {
    set(this.dashboard, `config.variables[${index}]`, variable);
  }

  reorderVariables(startIndex: number, endIndex: number) {
    const result = Array.from(get(this.dashboard, 'config.variables', []));
    const [removed] = result.splice(startIndex, 1);
    result.splice(endIndex, 0, removed);
    set(this.dashboard, `config.variables`, result);
  }

  deleteVariable(index: string) {
    pullAt(this.getVariables(), parseInt(index));
  }

  getVariableByIndex(index: string): Variable {
    return get(this.dashboard, `config.variables[${index}]`, {} as Variable);
  }

  sortPanels() {
    const panels = this.getPanels();
    panels.sort((a: PanelSetting, b: PanelSetting) => {
      if (a.grid?.y === b.grid?.y) {
        return a.grid?.x - b.grid?.x;
      }
      return a.grid?.y - b.grid?.y;
    });
  }

  private initDashboard(dashboard: Dashboard) {
    this.dashboard = dashboard;
    const panels = this.getPanels();
    const maxPanel = maxBy(panels, (panel: PanelSetting) => {
      if (!panel.id || panel.id < 0) {
        panel.id = this.assignPanelId();
      }
      // NOTE: set grid i here
      this.setPanelGridId(panel);
      return panel.id;
    });
    if (maxPanel) {
      this.panelSeq = maxPanel.id || 0;
    }
  }

  async loadDashbaord(dashboardId: string | null) {
    // reset panel seq when load dashboard
    this.panelSeq = 0;
    if (isEmpty(dashboardId)) {
      const panelId = this.assignPanelId();
      this.dashboard = {
        title: 'New Dashboard',
        config: {
          panels: [
            {
              title: 'Add panel',
              type: VisualizationAddPanelType,
              grid: { w: 12, h: 7, x: 0, y: 0, i: `${panelId}` }, // NOTE: must set i value(string)
              id: panelId,
            },
          ],
        },
      };
      return this.dashboard;
    }
    try {
      const dashboard = await DashboardSrv.getDashboard(`${dashboardId}`);
      this.initDashboard(dashboard);
    } catch (err) {
      console.warn('load dashobard error', err);
      Notification.error(ApiKit.getErrorMsg(err));
    }
    return this.dashboard;
  }

  async saveDashboard() {
    try {
      const dashboard = ObjectKit.removeUnderscoreProperties(toJS(this.dashboard));
      const panels = get(dashboard, 'config.panels', []);
      panels.forEach((panel: PanelSetting) => {
        // only keep x/y/w/h for grid
        panel.grid = pick(panel.grid, ['x', 'y', 'w', 'h']) as any;
      });
      this.sortPanels();
      // FIXME: remove unused field
      if (isEmpty(this.dashboard.uid)) {
        const uid = await DashboardSrv.createDashboard(dashboard);
        this.dashboard.uid = uid;
      } else {
        await DashboardSrv.updateDashboard(dashboard);
      }
      Notification.success('Save dashboard successfully!');
      return true;
    } catch (err) {
      console.warn('save dashobard error', err);
      Notification.error(ApiKit.getErrorMsg(err));
      return false;
    }
  }
}

export default new DashboardStore();
