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
import * as _ from 'lodash-es';
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
    const panels = _.get(this.dashboard, 'config.panels', []);
    return _.find(panels, { id: panelId });
  }

  clonePanel(panel: PanelSetting) {
    const newPanel: PanelSetting = _.cloneDeep(panel);
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
    const firstPanelType = _.get(this.dashboard, 'config.panels[0].type', null);
    if (firstPanelType === VisualizationAddPanelType) {
      // first panel is add panel widget, ignore this panel
      return;
    }
    if (_.has(this.dashboard, 'config.panels')) {
      _.set(this.dashboard, 'config.panels', _.concat(panel, this.dashboard.config?.panels));
    } else {
      _.set(this.dashboard, 'config.panels', [panel]);
    }
    this.sortPanels();
  }

  deletePanel(panel: PanelSetting) {
    const panels = _.get(this.dashboard, 'config.panels', []);
    const index = _.findIndex(panels, { id: panel.id });
    panels.splice(index, 1);
    this.sortPanels();
  }

  updatePanel(panel: PanelSetting) {
    const panels = _.get(this.dashboard, 'config.panels', []);
    const index = _.findIndex(panels, { id: panel.id });
    if (index >= 0) {
      panels[index] = panel;
      console.log('update pp', toJS(panel));
      this.sortPanels();
    }
  }

  updatePanelConfig(panel: any, config: any) {
    _.forIn(config, function (value, key) {
      _.set(panel, key, value);
    });
    // const panels = _.get(this.dashboard, 'config.panels', []);
    // const index = _.findIndex(panels, { id: panel.id });
    // _.set(this.dashboard, `config.panels[${index}]`, panel);
  }

  updateDashboardProps(values: any) {
    console.log(toJS(this.dashboard));
    this.dashboard = _.merge(this.dashboard, values);
  }

  addVariable(variable: any) {
    if (_.has(this.dashboard, 'config.variables')) {
      _.set(this.dashboard, 'config.variables', _.concat(this.dashboard.config?.variables, variable));
    } else {
      _.set(this.dashboard, 'config.variables', [variable]);
    }
  }

  sortPanels() {
    const panels = _.get(this.dashboard, 'config.panels', []);
    panels.sort((a: PanelSetting, b: PanelSetting) => {
      if (a.grid.y === b.grid.y) {
        return a.grid.x - b.grid.x;
      }
      return a.grid.y - b.grid.y;
    });
  }

  async saveDashboard() {
    try {
      const panels = _.get(this.dashboard, 'config.panels', []);
      panels.forEach((panel: PanelSetting) => {
        // only keep x/y/w/h for grid
        panel.grid = _.pick(panel.grid, ['x', 'y', 'w', 'h']);
      });
      this.sortPanels();
      // FIXME: remove unused field
      if (_.isEmpty(this.dashboard.uid)) {
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
