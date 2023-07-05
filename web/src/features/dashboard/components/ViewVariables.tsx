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
import { Card, Form, Input, useFormApi } from '@douyinfe/semi-ui';
import { IconSearchStroked } from '@douyinfe/semi-icons';
import React, { MutableRefObject, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { get, isEmpty, map, isArray, pick, includes, remove, debounce } from 'lodash-es';
import { Tracker, Variable, VariableHideType, VariableType } from '@src/types';
import './variables.scss';
import { useSearchParams } from 'react-router-dom';
import { useVariable } from '@src/hooks';
import { VariableContext } from '@src/contexts';
import { DatasourceStore } from '@src/stores';

const VariableSelect: React.FC<{
  variable: Variable;
  options: any[];
  loading?: boolean;
  remote?: boolean;
  onSearch?: (v: string) => void;
}> = (props) => {
  const { variable, options = [], loading, remote = false, onSearch } = props;
  const formApi = useFormApi();
  const dropdownVisible = useRef(false) as MutableRefObject<boolean>;
  const selectRef = useRef<any>();

  return (
    <Form.Select
      ref={selectRef}
      showClear
      loading={loading}
      key={variable.name}
      multiple={variable.multi}
      noLabel={variable.hide === VariableHideType.OnlyValue}
      label={variable.label}
      field={variable.name}
      remote={remote}
      filter
      optionList={options}
      onClear={() => formApi.submitForm()}
      innerTopSlot={
        <Input
          style={{ margin: 4, width: 'calc(100% - 8px)' }}
          suffix={<IconSearchStroked />}
          onChange={(value: string) => {
            selectRef.current.search(value);
          }}
        />
      }
      onChange={(_value) => {
        if (!variable.multi || !dropdownVisible.current) {
          formApi.submitForm();
        }
      }}
      onSearch={(v: string) => {
        if (onSearch) {
          onSearch(v);
        }
      }}
      onDropdownVisibleChange={(val) => {
        dropdownVisible.current = val;
        if (!val && variable.multi) {
          formApi.submitForm();
        }
      }}
    />
  );
};

/*
 * Custom variable based on setting.
 */
const CustomVariable: React.FC<{ variable: Variable }> = (props) => {
  const { variable } = props;

  return (
    <VariableSelect
      variable={variable}
      options={(variable.options || []).map((opt: any) => {
        return { label: opt.text, value: opt.value, showTick: false };
      })}
    />
  );
};

/*
 * Query variable based on setting.
 */
const QueryVariable: React.FC<{ variable: Variable; relatedVarNames: string[] }> = (props) => {
  const { variable, relatedVarNames } = props;
  const { variables } = useContext(VariableContext);
  const [input, setInput] = useState('');
  const { result, loading, refetch } = useVariable(variable, input);
  const formApi = useFormApi();
  const reloadValues = pick(variables, relatedVarNames || []);
  const reloadTracker = useRef() as MutableRefObject<Tracker<any>>;
  const search = debounce(setInput, 200);

  /**
   * initialize value
   */
  useMemo(() => {
    reloadTracker.current = new Tracker(reloadValues);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!isEmpty(relatedVarNames)) {
      if (reloadTracker.current.isChanged(reloadValues)) {
        reloadTracker.current.setNewVal(reloadValues);
        refetch();
      }
    }
  }, [refetch, reloadValues, relatedVarNames, formApi, variable]);

  /**
   * reset variable value if current value not in select values.
   */
  useEffect(() => {
    if (!loading) {
      const currentValue = formApi.getValue(variable.name);
      if (isEmpty(currentValue)) {
        return;
      }
      // check if need clean current selected value
      if (variable.multi) {
        // remove value that not contain select values.
        const removed = remove(currentValue, (v: string) => {
          return !includes(result, v);
        });
        if (!isEmpty(removed)) {
          formApi.setValue(variable.name, currentValue);
          formApi.submitForm();
        }
      } else {
        if (!includes(result, currentValue)) {
          formApi.setValue(variable.name, null);
          formApi.submitForm();
        }
      }
    }
  }, [result, formApi, variable, loading]);

  return (
    <VariableSelect
      variable={variable}
      onSearch={search}
      options={map(result, (r: string) => {
        return { value: r, label: r, showTick: false };
      })}
      loading={loading}
      remote={true}
    />
  );
};

/*
 * View the list of variables
 */
const ViewVariables: React.FC<{ className?: string }> = (props) => {
  const { className } = props;
  const { variables, definitions } = useContext(VariableContext);
  const [searchParams, setSearchParams] = useSearchParams();
  const formApi = useRef<any>();

  useEffect(() => {
    if (isEmpty(definitions) || !formApi.current) {
      return;
    }
    formApi.current.setValues(variables);
  }, [definitions, variables, formApi]);

  const findVariableNames = (v: Variable): string[] => {
    const query = v.query;
    if (isEmpty(query)) {
      return [];
    }
    // get datasource by uid
    const ds = DatasourceStore.getDatasource(query.datasource.uid);
    if (!ds) {
      return [];
    }
    return ds.api.findVariableNames(query);
  };

  if (isEmpty(definitions)) {
    return null;
  }

  return (
    <Card className={className} bodyStyle={{ padding: 6 }}>
      <Form
        labelPosition="inset"
        layout="horizontal"
        className="lin-variables"
        getFormApi={(api: any) => (formApi.current = api)}
        onSubmit={(values: any) => {
          // set variables to url params
          definitions.forEach((variable: Variable) => {
            const name = variable.name;
            const val = get(values, name);
            // first deleve old value
            searchParams.delete(name);
            if (!isEmpty(val)) {
              // if has selected, set new value
              if (variable.multi && isArray(val)) {
                val.forEach((v: string) => searchParams.append(name, v));
              } else {
                searchParams.set(name, val);
              }
            }
          });

          setSearchParams(searchParams);
        }}>
        {definitions.map((item: Variable) => {
          if (item.hide === VariableHideType.Hide) {
            return null;
          }
          switch (item.type) {
            case VariableType.Query:
              return <QueryVariable variable={item} key={item.name} relatedVarNames={findVariableNames(item)} />;
            case VariableType.Custom:
              return <CustomVariable variable={item} key={item.name} />;
          }
        })}
      </Form>
    </Card>
  );
};

export default ViewVariables;
