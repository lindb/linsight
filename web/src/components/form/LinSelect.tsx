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
import React, { MutableRefObject, useEffect, useMemo, useRef } from 'react';
import { Form, useFormApi, useFormState } from '@douyinfe/semi-ui';
import { isEmpty, upperFirst, debounce, pick } from 'lodash-es';
import { useRequest } from '@src/hooks';
import { ApiKit } from '@src/utils';
import { Notification } from '@src/components';
import { Tracker } from '@src/types';

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
  resetValue?: any;
  outerBottomSlot?: React.ReactNode;
  onFinished?: () => void;
}> = (props) => {
  const {
    field,
    resetValue,
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
    onFinished,
  } = props;
  const formState = useFormState();
  const formApi = useFormApi();
  const formValues = formState.values;
  const dropdownVisible = useRef(false) as MutableRefObject<boolean>;
  const searchInput = useRef('') as MutableRefObject<string>;
  const reloadValues = pick(formValues, reloadKeys || []);
  const reloadTracker = useRef() as MutableRefObject<Tracker<any>>;
  const lastLoad = useRef(new Date().getTime()) as MutableRefObject<number>;

  /**
   * initialize value
   */
  useMemo(() => {
    reloadTracker.current = new Tracker(reloadValues);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

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
      if (reloadTracker.current.isChanged(reloadValues)) {
        reloadTracker.current.setNewVal(reloadValues);
        formApi.setValue(field, resetValue || '');
        refetch();
      }
    }
  }, [refetch, reloadValues, field, formApi, resetValue, reloadKeys]);

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
        onClear={() => onFinished && onFinished()}
        onChange={() => {
          if (onFinished) {
            if (!multiple || !dropdownVisible.current) {
              // 1. signle select, if value change need trigger
              // 2. multiple select, if value change by result delete btn need trigger
              onFinished();
            }
          }
        }}
        onDropdownVisibleChange={(val) => {
          dropdownVisible.current = val;
          if (!val && onFinished) {
            onFinished();
          }
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
