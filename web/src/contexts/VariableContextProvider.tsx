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
import React, { createContext, MutableRefObject, useEffect, useMemo, useRef, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { SearchParamKeys, Tracker, Variable } from '@src/types';
import { isEmpty, set } from 'lodash-es';

export const VariableContext = createContext({
  variables: {},
  definitions: [] as Variable[],
  from: '',
  to: '',
  refreshTime: 0,
  refreshInterval: '',
  refresh: () => {},
});

const getValues = (variables: Variable[], searchParams: URLSearchParams): object => {
  const newValues = {};
  variables.forEach((variable: Variable) => {
    if (searchParams.has(variable.name)) {
      // FIXME: add multi/all logic
      if (variable.multi) {
        set(newValues, variable.name, searchParams.getAll(variable.name));
      } else {
        set(newValues, variable.name, searchParams.get(variable.name));
      }
    }
  });
  return newValues;
};

export const VariableContextProvider: React.FC<{ variables: Variable[]; children: React.ReactNode }> = (props) => {
  const { children, variables } = props;
  const [searchParams] = useSearchParams();
  const [valuesOfVariable, setValuesOfVariable] = useState(() => {
    return getValues(variables, searchParams);
  });
  const [refreshTime, setRefreshTime] = useState(0);
  const from = searchParams.get(SearchParamKeys.From) || '';
  const to = searchParams.get(SearchParamKeys.To) || '';
  const refreshInterval = searchParams.get(SearchParamKeys.Refresh) || '';
  const valuesTrackerRef = useRef() as MutableRefObject<Tracker<any>>;

  useMemo(() => {
    valuesTrackerRef.current = new Tracker<any>(null);
  }, []);

  useEffect(() => {
    if (isEmpty(variables)) {
      return;
    }
    const newValues = getValues(variables, searchParams);
    if (valuesTrackerRef.current.isChanged(newValues)) {
      valuesTrackerRef.current.setNewVal(newValues);
      setValuesOfVariable(newValues);
    }
  }, [searchParams, variables]);

  const refresh = () => {
    setRefreshTime(refreshTime + 1);
  };

  return (
    <VariableContext.Provider
      value={{
        definitions: variables || [],
        variables: valuesOfVariable,
        from,
        to,
        refreshTime,
        refresh,
        refreshInterval,
      }}>
      {children}
    </VariableContext.Provider>
  );
};
