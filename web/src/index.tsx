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
import React from 'react';
import ReactDOM from 'react-dom/client';
import { Route, Routes, createBrowserRouter, createRoutesFromElements, RouterProvider } from 'react-router-dom';
import { PlatformContextProvider } from '@src/contexts';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import en_US from '@douyinfe/semi-ui/lib/es/locale/source/en_US';
import { LocaleProvider } from '@douyinfe/semi-ui';
import '@src/styles/index.scss';
import App from '@src/App';
import Login from '@src/Login';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      cacheTime: 0,
    },
  },
});

const AppPage: React.FC = () => {
  return (
    <PlatformContextProvider>
      <div>
        <App />
      </div>
    </PlatformContextProvider>
  );
};

//https://stackoverflow.com/questions/75135147/react-router-dom-v6-useblocker
const router = createBrowserRouter(
  createRoutesFromElements(
    <Route
      path="/*"
      element={
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={<AppPage />} />
        </Routes>
      }
    />
  )
);

const container = document.getElementById('linsight') as HTMLElement;
const linSight = ReactDOM.createRoot(container!);
linSight.render(
  <QueryClientProvider client={queryClient}>
    <LocaleProvider locale={en_US}>
      <RouterProvider router={router} />
    </LocaleProvider>
  </QueryClientProvider>
);
