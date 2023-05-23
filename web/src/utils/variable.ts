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
import { OptionType, Variable } from '@src/types';
import { indexOf, set } from 'lodash-es';

/*
 * Variable if need multi select
 */
const isMulti = (variable: Variable): boolean => {
  return indexOf(variable.optionType, OptionType.Multi) >= 0;
};

/*
 * Set variable values from search params.
 */
const setVariableValues = (searchParams: URLSearchParams, variables: Variable[]) => {
  (variables || []).forEach((variable: Variable) => {
    if (searchParams.has(variable.name)) {
      if (isMulti(variable)) {
        set(variable, 'current.value', searchParams.getAll(variable.name));
      } else {
        set(variable, 'current.value', searchParams.get(variable.name));
      }
    }
  });
};

export default {
  setVariableValues,
  isMulti,
};
