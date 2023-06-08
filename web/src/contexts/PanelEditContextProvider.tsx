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
import React, { createContext, MutableRefObject, useMemo, useRef, useState } from 'react';
import { PanelSetting, Tracker } from '@src/types';
import { ObjectKit } from '@src/utils';
import { DashboardStore } from '@src/stores';
import { cloneDeep } from 'lodash-es';

/*
 * Context for panel editor
 */
export const PanelEditContext = createContext({
  panel: {} as PanelSetting,
  modifyPanel: (_panel: PanelSetting) => {},
});

/*
 * Context provider for each panel editor
 */
export const PanelEditContextProvider: React.FC<{ initPanel: PanelSetting; children: React.ReactNode }> = (props) => {
  const { initPanel = {}, children } = props;
  const [panel, setPanel] = useState(initPanel);
  const panelTracker = useRef() as MutableRefObject<Tracker<PanelSetting>>;
  useMemo(() => {
    panelTracker.current = new Tracker(initPanel);
  }, [initPanel]);
  /*
   * Modify panel options
   */
  const modifyPanel = (cfg: PanelSetting) => {
    const newPanel = cloneDeep(ObjectKit.merge(panel || {}, ObjectKit.removeUnderscoreProperties(cfg)));
    if (panelTracker.current.isChanged(newPanel)) {
      console.error('change panel data.......');
      panelTracker.current.setNewVal(newPanel);
      // NOTE: need modify dashboard's panel to trigger panel options modify event
      // because clone create new object, modify dashboard panel ref
      DashboardStore.updatePanel(newPanel);
      setPanel(newPanel);
    }
  };

  return (
    <PanelEditContext.Provider
      value={{
        panel,
        modifyPanel,
      }}>
      {children}
    </PanelEditContext.Provider>
  );
};
