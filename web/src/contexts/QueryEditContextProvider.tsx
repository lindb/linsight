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
import React, { createContext, useRef, useState } from 'react';
import { isEqual, cloneDeep } from 'lodash-es';
/*
 * Context for each query editor
 */
export const QueryEditContext = createContext({
  values: {} as object,
  setValues: (_values: object) => {},
});

/*
 * Context provider for each query editor
 */
export const QueryEditContextProvider: React.FC<{
  initValues?: object;
  onValuesChange?: (value: object) => void;
  children: React.ReactNode;
}> = (props) => {
  const { initValues = {}, children, onValuesChange } = props;
  const [values, setValues] = useState(initValues);
  const previous = useRef(cloneDeep(initValues));

  const modifyValues = (newValues: object) => {
    if (!isEqual(newValues, previous.current)) {
      previous.current = newValues;
      setValues(cloneDeep(newValues));

      if (onValuesChange) {
        onValuesChange(newValues);
      }
    }
  };

  return (
    <QueryEditContext.Provider
      value={{
        values,
        setValues: modifyValues,
      }}>
      {children}
    </QueryEditContext.Provider>
  );
};
