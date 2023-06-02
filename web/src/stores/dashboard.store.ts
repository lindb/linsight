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
import { DefaultColumns, PanelGridPos, RowPanelType, VisualizationAddPanelType } from '@src/constants';
import { DashboardSrv } from '@src/services';
import { Dashboard, GridPos, PanelSetting, Variable, Tracker } from '@src/types';
import { ApiKit, ObjectKit } from '@src/utils';
import { set, get, find, cloneDeep, has, concat, findIndex, forIn, merge, pick, isEmpty, pullAt } from 'lodash-es';
import { makeAutoObservable, toJS } from 'mobx';

class DashboardStore {
  private panelSeq = 0;
  private dashboardTracker: Tracker<Dashboard> = new Tracker<Dashboard>({});
  public dashboard: Dashboard = {};

  constructor() {
    makeAutoObservable(this);
  }

  private assignPanelId(): number {
    this.panelSeq++;
    return this.panelSeq;
  }

  getPanels(): PanelSetting[] {
    return get(this.dashboard, 'panels', []);
  }

  getPanel(panelId: number | undefined): PanelSetting | undefined {
    const panels = this.getPanels();
    return find(panels, { id: panelId });
  }

  collapseRow(row: PanelSetting, collapsed: boolean) {
    const cfg: PanelSetting = { collapsed: collapsed };
    if (collapsed) {
      if (isEmpty(row.panels)) {
        // if no child try to find children from dashboard's panels
        const panels = this.getPanels();
        const childIndexes: number[] = [];
        this.findPanelsForRow(row.id, (_child: PanelSetting, index: number) => [childIndexes.push(index)]);
        const children = pullAt(panels, ...childIndexes);
        cfg.panels = children;
      }
    } else {
      if (row.panels) {
        // has children, need put all children to dashboard's panels, and clean children
        const panels = this.getPanels();
        const index = findIndex(panels, { id: row.id });
        if (index >= 0) {
          const rowGridPos = this.getPanelGrid(row);
          const offsetOfChildren = rowGridPos.y + rowGridPos.h;
          let yMax = rowGridPos.y;
          let insertPos = index + 1;
          row.panels.forEach((child: PanelSetting) => {
            const childGridPos = this.getPanelGrid(child);
            childGridPos.y = offsetOfChildren;
            panels.splice(insertPos, 0, child);
            insertPos++;
            yMax = Math.max(yMax, childGridPos.y + childGridPos.h);
          });

          // re-calc panels' y after current row panel
          (panels.slice(insertPos) || []).forEach((child: PanelSetting) => {
            const childGridPos = this.getPanelGrid(child);
            childGridPos.y += yMax;
          });
        }
        // NOTE: clear panels for row panel
        cfg.panels = [];
      }
    }
    this.updatePanelConfig(row, cfg);
    // NOTE: must sort panels
    this.sortPanels();
  }

  clonePanel(panel: PanelSetting) {
    const newPanel: PanelSetting = cloneDeep(panel);
    if (!newPanel.gridPos || !panel.gridPos) {
      return;
    }
    //FIXME: remove other props?
    newPanel.id = this.assignPanelId();
    if (panel.gridPos.x + 2 * panel.gridPos.w <= DefaultColumns) {
      newPanel.gridPos.x += panel.gridPos.w;
    } else {
      newPanel.gridPos.y += panel.gridPos.y;
    }
    this.addPanel(newPanel);
    console.error(newPanel, toJS(this.dashboard));
  }

  createPanelConfig(newID: boolean = true): PanelSetting {
    const cfg: PanelSetting = {
      title: 'Add panel',
      type: VisualizationAddPanelType,
      gridPos: { w: 12, h: 8, x: 0, y: 0 },
    };
    if (newID) {
      cfg.id = this.assignPanelId();
    }
    return cfg;
  }

  addPanel(panel: PanelSetting) {
    const firstPanelType = get(this.dashboard, 'panels[0].type', null);
    if (firstPanelType === VisualizationAddPanelType) {
      // first panel is add panel widget, ignore this panel
      return;
    }
    this.setPanelGridId(panel);
    if (has(this.dashboard, 'panels')) {
      const panels = this.dashboard.panels || [];
      if (panel.type === VisualizationAddPanelType) {
        const newPanelHeight = this.getPanelGrid(panel).h;
        panels.forEach((child: PanelSetting) => {
          const childGridPos = this.getPanelGrid(child);
          childGridPos.y += newPanelHeight;
        });
      }
      set(this.dashboard, 'panels', concat(panel, panels));
    } else {
      set(this.dashboard, 'panels', [panel]);
    }
    this.sortPanels();
  }

  deleteRowAndChildren(row: PanelSetting) {
    const childIndexes: number[] = [];
    this.findPanelsForRow(row.id, (_child: PanelSetting, index: number) => [childIndexes.push(index)]);
    if (!isEmpty(childIndexes)) {
      // remove all children
      const panels = this.getPanels();
      pullAt(panels, ...childIndexes);
    }

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

  updateDashboardProps(values: any) {
    this.dashboard = merge(this.dashboard, values);
  }

  getVariables(): Variable[] {
    return get(this.dashboard, 'templating.list', []) as Variable[];
  }

  addVariable(variable: any) {
    const variables = this.getVariables();
    if (has(this.dashboard, 'templating.list')) {
      set(this.dashboard, 'templating.list', concat(variables, variable));
    } else {
      set(this.dashboard, 'templating.list', [variable]);
    }
  }

  updateVariable(index: string, variable: any) {
    set(this.dashboard, `templating.list[${index}]`, variable);
  }

  swapVariables(startIndex: number, endIndex: number) {
    const result = get(this.dashboard, 'templating.list', []) as Variable[];
    const [removed] = result.splice(startIndex, 1);
    result.splice(endIndex, 0, removed);
    set(this.dashboard, `templating.list`, result);
  }

  deleteVariable(index: string) {
    pullAt(this.getVariables(), parseInt(index));
  }

  getVariableByIndex(index: string): Variable {
    return get(this.dashboard, `templating.list[${index}]`, {} as Variable);
  }

  sortPanels() {
    const panels = this.getPanels();
    panels.sort((a: PanelSetting, b: PanelSetting) => {
      const aGridPos = this.getPanelGrid(a);
      const bGridPos = this.getPanelGrid(b);
      if (aGridPos.y === bGridPos.y) {
        return aGridPos.x - bGridPos.x;
      }
      return aGridPos.y - bGridPos.y;
    });
  }

  isDashboardChanged(): boolean {
    // need remove underscore temp props
    return this.dashboardTracker.isChanged(ObjectKit.removeUnderscoreProperties(this.dashboard));
  }

  async loadDashbaord(dashboardId: string | null) {
    // reset panel seq when load dashboard
    this.panelSeq = 0;
    if (isEmpty(dashboardId)) {
      const panelId = this.assignPanelId();
      this.dashboard = {
        title: 'New Dashboard',
        panels: [
          {
            title: 'Add panel',
            type: VisualizationAddPanelType,
            gridPos: { w: 12, h: 8, x: 0, y: 0, i: `${panelId}` }, // NOTE: must set i value(string)
            id: panelId,
          },
        ],
      };
      return this.dashboard;
    }
    try {
      const dashboard = await DashboardSrv.getDashboard(`${dashboardId}`);
      this.initDashboard(dashboard.dashboard);
    } catch (err) {
      console.warn('load dashobard error', err);
      Notification.error(ApiKit.getErrorMsg(err));
    }
    return this.dashboard;
  }

  async saveDashboard() {
    try {
      const dashboard = ObjectKit.removeUnderscoreProperties(toJS(this.dashboard));
      this.sortPanels();
      const panels = get(dashboard, 'panels', []);
      this.forEachAllPanels(panels, (panel: PanelSetting) => {
        // only keep x/y/w/h for grid
        panel.gridPos = pick(panel.gridPos, PanelGridPos) as any;
      });
      // FIXME: remove unused field
      if (isEmpty(this.dashboard.uid)) {
        const uid = await DashboardSrv.createDashboard(dashboard);
        dashboard.uid = uid;
      } else {
        await DashboardSrv.updateDashboard(dashboard);
      }
      this.setDashbaord(dashboard);
      Notification.success('Save dashboard successfully!');
      return true;
    } catch (err) {
      console.warn('save dashobard error', err);
      Notification.error(ApiKit.getErrorMsg(err));
      return false;
    }
  }
  private setDashbaord(dashboard: Dashboard) {
    this.dashboard = dashboard;
    this.dashboardTracker.setNewVal(dashboard);
  }

  /**
   * include panels of row
   */
  private forEachAllPanels(panels: PanelSetting[], handle: (panel: PanelSetting) => void) {
    if (isEmpty(panels)) {
      return;
    }
    panels.forEach((panel: PanelSetting) => {
      handle(panel);
      if (panel.type === RowPanelType) {
        this.forEachAllPanels(panel.panels || [], handle);
      }
    });
  }

  private initDashboard(dashboard: Dashboard) {
    this.dashboard = dashboard;
    const panels = this.getPanels();
    // first get max panel id
    let maxPanelId = 0;
    this.forEachAllPanels(panels, (panel: PanelSetting) => {
      maxPanelId = Math.max(maxPanelId, panel.id ?? 0);
    });
    this.panelSeq = maxPanelId;
    // set panel id if not set
    this.forEachAllPanels(panels, (panel: PanelSetting) => {
      if (!panel.id || panel.id < 0) {
        panel.id = this.assignPanelId();
      }
      // NOTE: set grid i here
      this.setPanelGridId(panel);
    });
    this.dashboardTracker.setNewVal(this.dashboard);
  }

  private getPanelGrid(panel: PanelSetting): GridPos {
    if (!panel.gridPos) {
      console.warn('return empty grid, please check panel no grid pos');
      return {} as GridPos;
    }
    return panel.gridPos;
  }

  private setPanelGridId(panel: PanelSetting) {
    // NOTE: make sure set grid i
    const gridPos = this.getPanelGrid(panel);
    gridPos.i = `${panel.id}`;
  }

  private findPanelsForRow(rowPanelId: number | undefined, callback: (child: PanelSetting, index: number) => void) {
    const panels = this.getPanels();
    const index = findIndex(panels, { id: rowPanelId });
    if (index < 0) {
      return;
    }
    for (let i = index + 1; i < panels.length; i++) {
      const panel = panels[i];
      if (panel.type === RowPanelType) {
        break;
      }
      callback(panel, i);
    }
  }
}

export default new DashboardStore();
