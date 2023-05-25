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
import { Card, Form, useFormApi } from '@douyinfe/semi-ui';
import { DashboardStore } from '@src/stores';
import React, { MutableRefObject, useEffect, useRef } from 'react';
import { get, set, isEmpty, map, isArray } from 'lodash-es';
import { observer } from 'mobx-react-lite';
import { Variable, VariableHideType, VariableType } from '@src/types';
import './variables.scss';
import { useSearchParams } from 'react-router-dom';
import { useVariable } from '@src/hooks';
import { VariableKit } from '@src/utils';
import { toJS } from 'mobx';

/*
 * Constant variable based on setting.
 */
const ConstantVariable: React.FC<{ variable: Variable }> = (props) => {
  const { variable } = props;

  return (
    <Form.Select
      showClear
      key={variable.name}
      noLabel={variable.hide === VariableHideType.OnlyValue}
      label={variable.label}
      field={variable.name}
      optionList={[
        { value: 'value1', label: 'value1' }, //FIXME: impl it
        { value: 'value2', label: 'value2' },
      ]}
    />
  );
};

/*
 * Query variable based on setting.
 */
const QueryVariable: React.FC<{ variable: Variable }> = (props) => {
  const { variable } = props;
  const { result, loading } = useVariable(variable, '');
  const formApi = useFormApi();
  const isMulti = VariableKit.isMulti(variable);
  const dropdownVisible = useRef(false) as MutableRefObject<boolean>;

  return (
    <Form.Select
      showClear
      key={variable.name}
      noLabel={variable.hide === VariableHideType.OnlyValue}
      multiple={isMulti}
      label={variable.label}
      field={variable.name}
      loading={loading}
      onClear={() => formApi.submitForm()}
      onChange={(_value) => {
        if (!isMulti || !dropdownVisible.current) {
          formApi.submitForm();
        }
      }}
      onDropdownVisibleChange={(val) => {
        dropdownVisible.current = val;
        if (!val && isMulti) {
          formApi.submitForm();
        }
      }}
      optionList={map(result, (r: string) => {
        return { value: r, label: r, showTick: false };
      })}
    />
  );
};

/*
 * View the list of variables
 */
const ViewVariables: React.FC<{ className?: string }> = (props) => {
  const { className } = props;
  const { dashboard } = DashboardStore;
  const [searchParams, setSearchParams] = useSearchParams();
  const variables: Variable[] = get(dashboard, 'config.variables', []);
  const formApi = useRef<any>();
  useEffect(() => {
    if (isEmpty(variables) || !formApi.current) {
      return;
    }
    const values = {};
    toJS(variables).forEach((variable: Variable) => {
      const val = get(variable, 'current.value');
      if (!isEmpty(val)) {
        set(values, variable.name, val);
      }
    });
    if (!isEmpty(values)) {
      formApi.current.setValues(values);
    }
  }, [variables, formApi]);

  if (isEmpty(variables)) {
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
          variables.forEach((variable: Variable) => {
            const name = variable.name;
            const val = get(values, name);
            // first deleve old value
            searchParams.delete(name);
            if (!isEmpty(val)) {
              // if has selected, set new value
              if (VariableKit.isMulti(variable) && isArray(val)) {
                val.forEach((v: string) => searchParams.append(name, v));
              } else {
                searchParams.set(name, val);
              }
            }
          });
          setSearchParams(searchParams);
        }}>
        {variables.map((item: Variable, index: number) => {
          if (item.hide === VariableHideType.Hide) {
            return null;
          }
          if (item.type === VariableType.Query) {
            return <QueryVariable variable={item} key={index} />;
          }
          return <ConstantVariable variable={item} key={index} />;
        })}
      </Form>
    </Card>
  );
};

export default observer(ViewVariables);
