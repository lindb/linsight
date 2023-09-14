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
import AutoSizer from 'react-virtualized-auto-sizer';
import { Dashboard } from '@src/components';
import { Observer } from 'mobx-react-lite';
import ViewVariables from './components/ViewVariables';

const View: React.FC = () => {
  return (
    <div>
      <ViewVariables className="variables" />
      <AutoSizer disableHeight>
        {({ width }) => {
          if (width == 0) {
            return null;
          }
          return (
            <div style={{ width: `${width}px` }}>
              <Observer>{() => <Dashboard />}</Observer>
            </div>
          );
        }}
      </AutoSizer>
    </div>
  );
};

export default View;
