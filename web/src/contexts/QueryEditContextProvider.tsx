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
import { cloneDeep } from 'lodash-es';
import { Query, Tracker } from '@src/types';
import { ObjectKit } from '@src/utils';
/*
 * Context for each query target editor
 */
export const QueryEditContext = createContext({
  target: {} as Query,
  modifyTarget: (_target: Query) => {},
});

/*
 * Context provider for each query target editor
 */
export const QueryEditContextProvider: React.FC<{
  initTarget?: Query;
  onTargetChange?: (value: Query) => void;
  children: React.ReactNode;
}> = (props) => {
  const { initTarget = {} as Query, children, onTargetChange } = props;
  const [target, setTarget] = useState(initTarget);
  const targetTracker = useRef() as MutableRefObject<Tracker<Query>>;
  useMemo(() => {
    targetTracker.current = new Tracker(initTarget);
  }, [initTarget]);

  const modifyTarget = (newTarget: Query) => {
    const newT = cloneDeep(ObjectKit.merge(target, newTarget));
    if (targetTracker.current.isChanged(newT)) {
      targetTracker.current.setNewVal(newT);
      setTarget(newT);

      if (onTargetChange) {
        onTargetChange(newT);
      }
    }
  };

  return (
    <QueryEditContext.Provider
      value={{
        target,
        modifyTarget,
      }}>
      {children}
    </QueryEditContext.Provider>
  );
};
