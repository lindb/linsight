import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter as Router, Route, Routes } from 'react-router-dom';
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

const container = document.getElementById('linsight') as HTMLElement;
const linSight = ReactDOM.createRoot(container!);
linSight.render(
  <QueryClientProvider client={queryClient}>
    <LocaleProvider locale={en_US}>
      <Router>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/*" element={<AppPage />} />
        </Routes>
      </Router>
    </LocaleProvider>
  </QueryClientProvider>
);
