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
import { Spin } from '@douyinfe/semi-ui';
import { Notification } from '@src/components';
import { PlatformSrv, UserSrv } from '@src/services';
import { DatasourceStore, MenuStore } from '@src/stores';
import { Bootdata, ThemeType } from '@src/types';
import { ApiKit } from '@src/utils';
import { useQuery } from '@tanstack/react-query';
import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';

export const PlatformContext = React.createContext({
  boot: {} as any,
  theme: ThemeType.Default,
  collapsed: false,
  toggleTheme: () => {},
  toggleCollapse: () => {},
});

export const PlatformContextProvider: React.FC<{ children?: React.ReactNode }> = (props) => {
  const { children } = props;
  const [boot, setBoot] = useState<Bootdata | any>({});
  const [collapsed, setCollapsed] = useState(true);
  const [theme, setTheme] = useState<ThemeType>(ThemeType.Default);
  const navigate = useNavigate();
  const [isLoading, setIsLoading] = useState(true);
  const { data, isError, error } = useQuery(['bootstrap'], async () => {
    // load all features
    const moduels = import.meta.glob(['../features/*/module.ts', '../plugins/**/module.ts']);
    for (const key in moduels) {
      await moduels[key]();
    }
    return PlatformSrv.boot();
  });

  useEffect(() => {
    if (data) {
      MenuStore.setMenus(data.navTree);
      DatasourceStore.setDatasources(data.datasources);
      setTheme(data.user.preference?.theme || ThemeType.Default);
      setCollapsed(data.user.preference?.collapsed || true);
      setBoot(data);
      setIsLoading(false);
    }
  }, [data, isError]);

  useEffect(() => {
    if (theme === ThemeType.Dark) {
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
    if (isError) {
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
        <div style={{ width: '100%', textAlign: 'center', marginTop: 300 }}>
          <Spin size="large" />
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
      }}>
      {renderContent()}
    </PlatformContext.Provider>
  );
};
