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
import React, { MutableRefObject, useCallback, useEffect, useRef } from 'react';
import { Form, useFieldState, useFormApi } from '@douyinfe/semi-ui';
import { isEmpty, isEqual, upperFirst, concat, debounce } from 'lodash-es';
import { useRequest } from '@src/hooks';
import { ApiKit } from '@src/utils';
import { Notification } from '@src/components';

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
    showClear,
    filter,
    remote,
    prefix,
    label,
    placeholder,
    style,
    labelPosition,
    visible,
    loader,
    reloadKeys,
    rules,
    clearKeys,
    outerBottomSlot,
    onChange,
  } = props;
  const formApi = useFormApi();
  const { value } = useFieldState(field);
  const show = visible ? visible() : true;

  const searchInput = useRef('') as MutableRefObject<string>;
  const dropdownVisible = useRef() as MutableRefObject<boolean>;
  const previousValue = useRef() as MutableRefObject<any>;
  const previousCascade = useRef({}) as MutableRefObject<any>;
  const valueTriggerURL = useRef() as MutableRefObject<boolean>; // mark value trigger url modify

  const isEnable = (): boolean => {
    console.log('xxxxkkkkkkk.............');
    return false;
  };
  const { result, loading, error, refetch } = useRequest(
    [field, loader],
    () => {
      parseInt('abc');
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
    if (show) {
      if (!isEmpty(reloadKeys)) {
        console.log('reload key.....', reloadKeys);
        refetch();
        // if (!isEqual(previousKeys.current, values)) {
        //   refetch();
        //   previousKeys.current = values;
        // }
      }
    }
  }, [refetch, reloadKeys, show]);

  useEffect(() => {
    if (valueTriggerURL.current) {
      valueTriggerURL.current = false;
      return;
    }
    if (show) {
      // const value = get(params, field);
      if (!isEqual(value, previousValue.current)) {
        let finalVal = undefined;
        if (!isEmpty(value)) {
          finalVal = multiple ? concat([], value) : value;
        }
        formApi.setValue(field, finalVal);
        previousValue.current = finalVal;
      }
    }
  }, [show, field, multiple, formApi]);

  if (!show) {
    return null;
  }

  // lazy remote search when user input.
  const search = debounce(refetch, 200);
  console.log('rrrrrrr.......', result);

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
          if (!val) {
            formApi.submitForm();
          }
        }}
        // onFocus={() => {
        //   if (error) {
        //     // if previous fetch failure, retry when focus
        //     refetch();
        //   }
        // }}
      />
    </>
  );
};

export default LinSelect;
