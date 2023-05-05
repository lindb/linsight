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
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
      },
    },
  },
  // base: '/',
  plugins: [
    react({
      include: '**/*.jsx',
    }),
  ],
  define: {
    'process.env': {},
    // global: '({})',
  },
  css: {
    preprocessorOptions: {
      scss: {
        javascriptEnabled: true,
      },
    },
  },
  resolve: {
    alias: [
      {
        find: /~(.+)/,
        replacement: path.resolve(__dirname, 'node_modules/$1'),
      },
      { find: '@src', replacement: path.resolve(__dirname, 'src') },
    ],
  },
  build: {
    outDir: 'static',
  },
});
