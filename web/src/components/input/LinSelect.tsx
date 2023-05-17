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
import React, { MutableRefObject, useEffect, useRef } from 'react';
import { Form, useFormApi, useFormState } from '@douyinfe/semi-ui';
import { isEmpty, isEqual, upperFirst, debounce, pick } from 'lodash-es';
import { useRequest } from '@src/hooks';
import { ApiKit } from '@src/utils';
import { Notification } from '@src/components';

const RELOAD_TIME = 5 * 60 * 1000;

/**
 * LinSelect implements remote load select component.
 */
const LinSelect: React.FC<{
  field: string;
  cascade?: string[];
  multiple?: boolean;
  showClear?: boolean;
  filter?: boolean;
  remote?: boolean;
  prefix?: React.ReactNode;
  label?: React.ReactNode;
  placeholder?: React.ReactNode;
  style?: React.CSSProperties;
  labelPosition?: 'top' | 'left' | 'inset';
  visible?: () => boolean;
  loader?: (input?: string) => any;
  reloadKeys?: string[];
  rules?: any[];
  clearKeys?: string[];
  outerBottomSlot?: React.ReactNode;
  onChange?: (value: string | number | any[] | Record<string, any>) => void;
}> = (props) => {
  const {
    field,
    cascade,
    multiple,
    showClear = true,
    filter = true,
    remote = true,
    prefix,
    label,
    placeholder,
    style,
    labelPosition = 'inset',
    loader,
    reloadKeys,
    rules,
    outerBottomSlot,
    onChange,
  } = props;
  const formState = useFormState();
  const formApi = useFormApi();
  const formValues = formState.values;

  const searchInput = useRef('') as MutableRefObject<string>;
  const dropdownVisible = useRef() as MutableRefObject<boolean>;
  const previousValuesOfReloadKeys = useRef(pick(formValues, reloadKeys || [])) as MutableRefObject<any>;
  const lastLoad = useRef(new Date().getTime()) as MutableRefObject<number>;

  const isEnable = (): boolean => {
    return true;
  };
  const { result, loading, error, refetch } = useRequest(
    [field, loader],
    () => {
      lastLoad.current = new Date().getTime();
      return loader ? loader(searchInput.current || '') : [];
    },
    {
      enabled: isEnable(),
      onError: (err: any) => {
        Notification.error(ApiKit.getErrorMsg(err));
      },
    }
  );

  useEffect(() => {
    if (!isEmpty(reloadKeys)) {
      const valuesOfReloadKeys = pick(formValues, reloadKeys || []);
      if (!isEqual(previousValuesOfReloadKeys.current, valuesOfReloadKeys)) {
        previousValuesOfReloadKeys.current = valuesOfReloadKeys;
        formApi.setValue(field, '');
        refetch();
      }
    }
  }, [refetch, reloadKeys, formValues, field, formApi]);

  // lazy remote search when user input.
  const search = debounce(refetch, 200);

  return (
    <>
      <Form.Select
        rules={rules}
        field={field}
        multiple={multiple}
        showClear={showClear}
        filter={filter}
        remote={remote}
        prefix={prefix}
        placeholder={placeholder}
        optionList={result}
        labelPosition={labelPosition || 'inset'}
        label={label || upperFirst(field)}
        style={style}
        loading={loading}
        outerBottomSlot={outerBottomSlot}
        onSearch={(input: string) => {
          if (remote && input !== searchInput.current) {
            searchInput.current = input;
            search();
          }
        }}
        onDropdownVisibleChange={(val) => {
          dropdownVisible.current = val;
          // TODO: need add change form submit logic
          // if (!val && onChange) {
          //   onChange(val);
          //   // formApi.submitForm();
          // }
        }}
        onFocus={() => {
          if (error || new Date().getTime() - lastLoad.current >= RELOAD_TIME) {
            // 1. if previous fetch failure, retry when focus
            // 2. last fetch time > RELOAD_TIME
            refetch();
          }
        }}
      />
    </>
  );
};

export default LinSelect;
