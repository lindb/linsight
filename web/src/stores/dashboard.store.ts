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
import { DefaultColumns, VisualizationAddPanelType } from '@src/constants';
import { DashboardSrv } from '@src/services';
import { Dashboard, PanelSetting } from '@src/types';
import { ApiKit } from '@src/utils';
import { set, get, find, cloneDeep, has, concat, findIndex, forIn, merge, pick, isEmpty, pullAt } from 'lodash-es';
import { makeAutoObservable, toJS } from 'mobx';

class DashboardStore {
  public dashboard: Dashboard = {};

  constructor() {
    makeAutoObservable(this);
  }

  setDashboard(dashboard: Dashboard) {
    console.log('set dashboard.........', dashboard);
    this.dashboard = dashboard;
  }

  getPanel(panelId: any): PanelSetting | undefined {
    console.log('xxxxxxxx get panel', panelId, toJS(this.dashboard));
    const panels = get(this.dashboard, 'config.panels', []);
    return find(panels, { id: panelId });
  }

  clonePanel(panel: PanelSetting) {
    const newPanel: PanelSetting = cloneDeep(panel);
    //FIXME: remove other props?
    newPanel.id = `${new Date().getTime()}`;
    if (panel.grid?.x + 2 * panel.grid?.w <= DefaultColumns) {
      newPanel.grid.x += panel.grid.w;
    } else {
      newPanel.grid.y += panel.grid.y;
    }
    this.addPanel(newPanel);
  }

  addPanel(panel: PanelSetting) {
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

  deletePanel(panel: PanelSetting) {
    const panels = get(this.dashboard, 'config.panels', []);
    const index = findIndex(panels, { id: panel.id });
    panels.splice(index, 1);
    this.sortPanels();
  }

  updatePanel(panel: PanelSetting) {
    const panels = get(this.dashboard, 'config.panels', []);
    const index = findIndex(panels, { id: panel.id });
    if (index >= 0) {
      panels[index] = panel;
      console.log('update pp', toJS(panel));
      this.sortPanels();
    }
  }

  updatePanelConfig(panel: any, config: any) {
    forIn(config, function (value, key) {
      set(panel, key, value);
    });
    // const panels = _.get(this.dashboard, 'config.panels', []);
    // const index = _.findIndex(panels, { id: panel.id });
    // _.set(this.dashboard, `config.panels[${index}]`, panel);
  }

  updateDashboardProps(values: any) {
    console.log(toJS(this.dashboard));
    this.dashboard = merge(this.dashboard, values);
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
    pullAt(get(this.dashboard, 'config.variables', []), parseInt(index));
  }

  getVariableByIndex(index: string) {
    return get(this.dashboard, `config.variables[${index}]`, {});
  }

  sortPanels() {
    const panels = get(this.dashboard, 'config.panels', []);
    panels.sort((a: PanelSetting, b: PanelSetting) => {
      if (a.grid.y === b.grid.y) {
        return a.grid.x - b.grid.x;
      }
      return a.grid.y - b.grid.y;
    });
  }

  async saveDashboard() {
    try {
      const panels = get(this.dashboard, 'config.panels', []);
      panels.forEach((panel: PanelSetting) => {
        // only keep x/y/w/h for grid
        panel.grid = pick(panel.grid, ['x', 'y', 'w', 'h']);
      });
      this.sortPanels();
      // FIXME: remove unused field
      if (isEmpty(this.dashboard.uid)) {
        await DashboardSrv.createDashboard(this.dashboard);
      } else {
        await DashboardSrv.updateDashboard(this.dashboard);
      }
      Notification.success('Save dashboard successfully!');
    } catch (err) {
      Notification.error(ApiKit.getErrorMsg(err));
    }
  }
}

export default new DashboardStore();
