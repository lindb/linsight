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
import { Loading, Notification } from '@src/components';
import { useRequest } from '@src/hooks';
import { PlatformSrv, UserSrv } from '@src/services';
import { DatasourceStore, MenuStore } from '@src/stores';
import { Bootdata, FormatRepositoryInst, ThemeType } from '@src/types';
import { ApiKit } from '@src/utils';
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';

/*
 * Global root context
 */
export const PlatformContext = React.createContext({
  boot: {} as any,
  theme: ThemeType.Default,
  collapsed: false,
  toggleTheme: () => {},
  toggleCollapse: () => {},
  sync: () => {},
});

/*
 * Global root platform context provider.
 */
export const PlatformContextProvider: React.FC<{ children?: React.ReactNode }> = (props) => {
  const { children } = props;
  const [boot, setBoot] = useState<Bootdata | any>({});
  const [collapsed, setCollapsed] = useState(true);
  const [theme, setTheme] = useState<ThemeType>(ThemeType.Default);
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(true);
  const { result, error, refetch } = useRequest(['bootstrap'], async () => {
    // load all features
    import.meta.glob(['../features/*/module.ts', '../plugins/**/module.ts'], { eager: true });
    // after load plugins moduels build format tree datasources
    FormatRepositoryInst.buildTree();
    return PlatformSrv.boot();
  });

  useEffect(() => {
    const handleDarkModeChange = (event: any) => {
      if (theme !== ThemeType.System) {
        return;
      }
      if (event.matches) {
        document.body.setAttribute('theme-mode', 'dark');
      } else {
        document.body.removeAttribute('theme-mode');
      }
    };
    let darkModeMedia: MediaQueryList;
    if (window.matchMedia) {
      // watch os system theme change
      darkModeMedia = window.matchMedia('(prefers-color-scheme: dark)');
      darkModeMedia.addEventListener('change', handleDarkModeChange);
    }
    return () => {
      if (darkModeMedia) {
        darkModeMedia.removeEventListener('change', handleDarkModeChange);
      }
    };
  }, [theme]);

  useEffect(() => {
    if (result) {
      MenuStore.setMenus(result.navTree);
      DatasourceStore.setDatasources(result.datasources);
      setTheme(result.user.preference?.theme || ThemeType.Default);
      setCollapsed(result.user.preference?.collapsed);
      setBoot(result);
      setIsLoading(false);
    }
  }, [result, error]);

  useEffect(() => {
    const isDark =
      theme === ThemeType.Dark ||
      (theme === ThemeType.System && window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches);
    if (isDark) {
      document.body.setAttribute('theme-mode', 'dark');
    } else {
      document.body.removeAttribute('theme-mode');
    }
  }, [theme]);

  const handleToggleTheme = () => {
    setTheme((t: ThemeType) => {
      let theme = ThemeType.Default;
      switch (t) {
        case ThemeType.Dark:
          theme = ThemeType.Light;
          break;
        case ThemeType.Light:
        default:
          theme = ThemeType.Dark;
          break;
      }
      const preference = boot.user.preference || {};
      preference.theme = theme;
      UserSrv.savePreference(preference);
      return theme;
    });
  };

  const handleToggleCollapsed = () => {
    setCollapsed(!collapsed);
  };

  const renderContent = () => {
    if (error) {
      const httpCode = ApiKit.getErrorCode(error);
      if (httpCode == 403) {
        navigate('/login');
      } else {
        Notification.error(ApiKit.getErrorMsg(error));
      }
      return null;
    }
    if (isLoading) {
      return (
        <div
          style={{ width: '100%', height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <Loading />
        </div>
      );
    }
    return children;
  };

  return (
    <PlatformContext.Provider
      value={{
        boot,
        theme,
        collapsed,
        toggleTheme: handleToggleTheme,
        toggleCollapse: handleToggleCollapsed,
        sync: refetch,
      }}>
      {renderContent()}
    </PlatformContext.Provider>
  );
};
