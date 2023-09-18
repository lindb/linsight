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
import React, { createContext, MutableRefObject, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { PanelSetting, Query, Tracker } from '@src/types';
import { ObjectKit } from '@src/utils';
import { DashboardStore, DatasourceStore } from '@src/stores';
import { cloneDeep, get, isEmpty, unset } from 'lodash-es';
import { MixedDatasource } from '@src/constants';
import { useSearchParams } from 'react-router-dom';

/*
 * Context for panel editor
 */
export const PanelEditContext = createContext({
  panel: {} as PanelSetting,
  modifyPanel: (_panel: PanelSetting, _overwrite?: boolean) => {},
});

/*
 * Context provider for each panel editor
 */
export const PanelEditContextProvider: React.FC<{
  urlBind?: string;
  initPanel: PanelSetting;
  children: React.ReactNode;
}> = (props) => {
  const { urlBind, initPanel = {}, children } = props;
  const [searchParams, setSearchParams] = useSearchParams();

  const getOptions = useCallback(
    (key: string) => {
      const defaultDS = DatasourceStore.getDefaultDatasource();
      const defaultPanelCfg = defaultDS?.plugin.getDefaultParams();
      const options = `${searchParams.get(key)}`;
      if (!options || isEmpty(options)) {
        return defaultPanelCfg;
      }
      try {
        const panel = JSON.parse(options);
        if (!panel) {
          return defaultPanelCfg;
        }
        return panel;
      } catch (err) {
        console.warn('parse metric explore error', err);
      }
      return defaultPanelCfg;
    },
    [searchParams]
  );

  const getInitPanel = useCallback(() => {
    if (!isEmpty(urlBind)) {
      return getOptions(`${urlBind}`);
    }
    return initPanel;
  }, [urlBind, getOptions, initPanel]);

  const panelToString = (p: PanelSetting): string => {
    return JSON.stringify(p, (_key: string, value: any) => {
      if (isEmpty(value)) return undefined;
      return value;
    });
  };

  const [panel, setPanel] = useState(null);
  const panelTracker = useRef() as MutableRefObject<Tracker<PanelSetting>>;
  const urlTracker = useRef() as MutableRefObject<Tracker<string>>;

  useMemo(() => {
    const init = ObjectKit.cleanEmptyProperties(getInitPanel());
    panelTracker.current = new Tracker(init);
    urlTracker.current = new Tracker(panelToString(init));
    setPanel(init);
  }, [getInitPanel]);

  useEffect(() => {
    if (!isEmpty(urlBind)) {
      const panel = getOptions(`${urlBind}`);
      setPanel(panel);
    }
  }, [urlBind, searchParams, getOptions]);

  /*
   * Modify panel options
   */
  const modifyPanel = (cfg: PanelSetting, overwrite?: boolean) => {
    let newPanel = ObjectKit.cleanEmptyProperties(
      cloneDeep(ObjectKit.merge(panel || {}, ObjectKit.removeUnderscoreProperties(cfg)))
    );
    if (overwrite || panelTracker.current.isChanged(newPanel)) {
      const newPanelDatasource = get(newPanel, 'datasource');
      const newPanelDatasourceUID = get(newPanel, 'datasource.uid');
      const oldPanelDatasourceUID = get(panelTracker.current.getVal(), 'datasource.uid');

      if (
        DatasourceStore.getDatasourceCategory(`${newPanelDatasourceUID}`) !==
        DatasourceStore.getDatasourceCategory(`${oldPanelDatasourceUID}`)
      ) {
        const ds = DatasourceStore.getDatasource(newPanelDatasourceUID);
        newPanel = ds?.plugin.getDefaultParams();
        newPanel.datasource = newPanelDatasource;
      }
      console.error('change panel data.......', newPanel, newPanelDatasourceUID);

      if (newPanelDatasourceUID !== MixedDatasource) {
        // if panel's datasource not mixed, need clean all targets' datasource, use panel datasource
        (newPanel.targets || []).forEach((target: Query) => {
          unset(target, 'datasource');
        });
      }

      panelTracker.current.setNewVal(newPanel);
      // NOTE: need modify dashboard's panel to trigger panel options modify event
      // because clone create new object, modify dashboard panel ref
      DashboardStore.updatePanel(newPanel);
      if (isEmpty(urlBind)) {
        setPanel(newPanel);
      } else {
        const p = panelToString(newPanel);
        if (urlTracker.current.isChanged(p)) {
          urlTracker.current.setNewVal(p);
          searchParams.set(`${urlBind}`, p);
          setSearchParams(searchParams);
        }
      }
    }
  };

  return (
    <PanelEditContext.Provider
      value={{
        panel: panel || {},
        modifyPanel,
      }}>
      {children}
    </PanelEditContext.Provider>
  );
};
