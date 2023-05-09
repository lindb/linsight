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
import { isEmpty, isEqual, upperFirst, concat, get, pick, debounce } from 'lodash-es';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@src/hooks';
import { URLStore } from '@src/stores';
import { ApiKit } from '@src/utils';
import { Notification } from '@src/components';

/**
 * LinSelect implements url binding select component.
 */
const LinSelect: React.FC<{
  field: string;
  defaultValue?: string | string[];
  multiple?: boolean;
  showClear?: boolean;
  filter?: boolean;
  remote?: boolean;
  bind?: boolean;
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
    defaultValue,
    multiple,
    showClear,
    filter,
    remote,
    bind,
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
  const fieldApi = useFormApi();
  const { value } = useFieldState(field);
  const params = useParams([field, ...(reloadKeys || [])]);
  const show = visible ? visible() : true;

  const searchInput = useRef('') as MutableRefObject<string>;
  const dropdownVisible = useRef() as MutableRefObject<boolean>;
  const previousValue = useRef() as MutableRefObject<any>;
  const previousKeys = useRef({}) as MutableRefObject<any>;
  const urlTriggerValue = useRef() as MutableRefObject<boolean>; // mark url triger value modify
  const valueTriggerURL = useRef() as MutableRefObject<boolean>; // mark value trigger url modify
  const { data, isInitialLoading, isFetching, isError, refetch } = useQuery(
    [field, loader],
    () => {
      return loader ? loader(searchInput.current || '') : [];
    },
    {
      enabled: show,
      onError: (err) => {
        Notification.error(ApiKit.getErrorMsg(err));
      },
    }
  );

  const changeURLParams = useCallback(() => {
    if (bind && !dropdownVisible.current && !isEqual(value, previousValue.current)) {
      valueTriggerURL.current = true;
      previousValue.current = value;
      // change url params after dropdown hidden
      if (value) {
        URLStore.changeURLParams({
          params: { [field]: value },
          needDelete: clearKeys || [],
        });
      } else {
        URLStore.changeURLParams({
          needDelete: [field],
        });
      }
      if (onChange) {
        onChange(value);
      }
    }
  }, [clearKeys, bind, value, field, onChange]);

  useEffect(() => {
    if (urlTriggerValue.current) {
      urlTriggerValue.current = false;
      return;
    }
    changeURLParams();
  }, [changeURLParams, clearKeys, onChange, value]);

  useEffect(() => {
    if (show) {
      if (!isEmpty(reloadKeys)) {
        console.log('reload key.....', reloadKeys);
        if (!bind) {
          refetch();
        } else {
          const values = pick(params, reloadKeys || []);
          if (!isEqual(previousKeys.current, values)) {
            refetch();
            previousKeys.current = values;
          }
        }
      }
    }
  }, [params, refetch, reloadKeys, show, bind]);

  useEffect(() => {
    if (valueTriggerURL.current) {
      valueTriggerURL.current = false;
      return;
    }
    if (show) {
      const value = get(params, field);
      if (!isEqual(value, previousValue.current)) {
        urlTriggerValue.current = true;
        let finalVal = undefined;
        if (!isEmpty(value)) {
          finalVal = multiple ? concat([], value) : value;
        }
        fieldApi.setValue(field, finalVal);
        previousValue.current = finalVal;
      }
    }
  }, [params, show, field, multiple, fieldApi]);

  /*
   * modify default value
   */
  useEffect(() => {
    if (!isEmpty(defaultValue)) {
      URLStore.changeDefaultParams({ [field]: defaultValue });
    } else {
      // FIXME: need delete params from url
      // URLStore.deleteDefaultParams([field]);
    }
  }, [defaultValue, fieldApi, field]);

  if (!show) {
    return null;
  }

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
        optionList={data}
        labelPosition={labelPosition || 'inset'}
        label={label || upperFirst(field)}
        style={style}
        loading={isInitialLoading || isFetching}
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
            changeURLParams();
          }
        }}
        onFocus={() => {
          if (isError) {
            // if previous fetch failure, retry when focus
            refetch();
          }
        }}
      />
    </>
  );
};

export default LinSelect;
