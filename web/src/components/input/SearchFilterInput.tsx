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
import React, { MutableRefObject, useEffect, useRef, useState } from 'react';
import { Checkbox, CheckboxGroup, Input, List, Popover, Tag, TagInput, Typography } from '@douyinfe/semi-ui';
import { IconSearchStroked } from '@douyinfe/semi-icons';
import { debounce, filter, find, get, isEmpty, join, trim } from 'lodash-es';
import './search-filter-input.scss';
import { useRequest } from '@src/hooks';
import { useSearchParams } from 'react-router-dom';

const { Text } = Typography;

export interface AttributeProps {
  type: 'input' | 'select';
  value: string;
  label?: string;
  multiple?: boolean;
  remote?: (prefix: string) => Promise<string[]>;
  options?: {
    value: any;
    label: string;
  }[];
  selectedValue?: string | string[];
}

export interface SearchFilterInputProps {
  placeholder?: string;
  attributes?: AttributeProps[];
  values?: any;
}

const MultipleSelect: React.FC<{
  values?: string[];
  type: 'input' | 'select';
  value: string;
  label?: string;
  multiple?: boolean;
  options?: any[];
  remote?: (prefix: string) => Promise<string[]>;
  onChange: (values: any[]) => void;
}> = (props) => {
  const { remote, values, multiple, options, value, onChange } = props;
  const [searchInput, setSearchInput] = useState('');
  const { result, loading } = useRequest(['load_att_values', searchInput], () => {
    if (remote) {
      return remote(searchInput);
    }
    return [];
  });
  if (!multiple) {
    return (
      <List
        className="suggest-list"
        dataSource={options || []}
        split={false}
        size="small"
        loading={loading}
        renderItem={(item) => (
          <List.Item
            className="list-item"
            onClick={() => {
              onChange(item);
            }}>
            {item.label}
          </List.Item>
        )}
      />
    );
  }

  const search = debounce(setSearchInput, 200);

  return (
    <div>
      <div key="search" style={{ padding: 8 }}>
        <Input key="search" prefix={<IconSearchStroked />} onChange={(v: string) => search(v)} />
      </div>
      <CheckboxGroup
        key="list"
        defaultValue={get(find(values, { label: value }), 'value')}
        onChange={(val: any[]) => {
          onChange(val);
        }}>
        <List
          className="suggest-list"
          dataSource={result || []}
          split={false}
          size="small"
          loading={loading}
          renderItem={(item) => (
            <List.Item style={{ padding: '4px 8px' }}>
              <Checkbox value={item}>{item}</Checkbox>
            </List.Item>
          )}
        />
      </CheckboxGroup>
    </div>
  );
};

export const SearchFilterInput: React.FC<SearchFilterInputProps> = (props) => {
  const { placeholder, attributes } = props;
  const [searchParams, setSearchParams] = useSearchParams();
  const [visible, setVisible] = useState(false);
  const [rePosKey, setRePosKey] = useState(0);
  const [inputValue, setInputValue] = useState('');
  const [pendingEdit, setPendingEdit] = useState<any>(null);
  const tagInputRef = useRef<any>();
  const refDiv = useRef() as MutableRefObject<HTMLDivElement>;
  const [tags, setTags] = useState<any[]>([]);

  useEffect(() => {
    const tags: any[] = [];
    attributes?.forEach((att: AttributeProps) => {
      if (!searchParams.has(att.value)) {
        return;
      }
      if (att.multiple) {
        tags.push({ label: att.value, value: searchParams.getAll(att.value) });
      } else {
        const options = att.options;
        const v = `${searchParams.get(att.value)}`;
        if (options) {
          tags.push({ label: att.value, value: get(find(options, { value: v }), 'label', '') });
        } else {
          tags.push({ label: att.value, value: v });
        }
      }
    });
    setTags(tags);
  }, [attributes, searchParams]);

  const getSearchInputPos = (): number => {
    const input = refDiv.current.querySelector('input');
    if (!input) {
      return -1;
    }
    const rect = input.getBoundingClientRect();
    return rect.x - refDiv.current.getBoundingClientRect().x;
  };

  useEffect(() => {
    if (visible) {
      setRePosKey(getSearchInputPos());
    } else {
      if (!pendingEdit) {
        return;
      }
      if (pendingEdit.type === 'select') {
        setInputValue('');
        if (isEmpty(pendingEdit)) {
          return;
        }
        if (pendingEdit.multiple) {
          searchParams.delete(pendingEdit.value);
          pendingEdit.selectedValue.forEach((v: string) => {
            searchParams.append(pendingEdit.value, v);
          });
        } else {
          searchParams.set(pendingEdit.value, pendingEdit.selectedValue.value);
        }
        setSearchParams(searchParams);
        setPendingEdit(null);
      }
    }
  }, [visible, pendingEdit, searchParams, setSearchParams]);

  const buildTagInputValue = (att: AttributeProps, value: any): string => {
    let v = `${att.label}:`;
    if (att.type === 'select') {
      if (att.multiple) {
        v += join(value, '|');
      } else {
        v += get(value, 'label', value);
      }
    } else {
      v += `${value}`;
    }
    return v;
  };

  const renderSuggests = () => {
    if (pendingEdit) {
      if (pendingEdit.type === 'select') {
        return (
          <MultipleSelect
            values={tags}
            {...pendingEdit}
            onChange={(values: any[]) => {
              pendingEdit.selectedValue = values;
              const value = buildTagInputValue(pendingEdit, values);
              setInputValue(value);
              if (!pendingEdit.multiple) {
                setVisible(false);
              }
            }}
          />
        );
      }
    }
    const suggest = filter(attributes || [], (a: AttributeProps) => {
      return !find(tags, { label: a.value });
    });
    if (isEmpty(suggest)) {
      return null;
    }
    return (
      <List
        className="suggest-list"
        split={false}
        size="small"
        dataSource={suggest}
        renderItem={(att: any) => {
          return (
            <List.Item
              className="list-item"
              onClick={() => {
                if (pendingEdit) {
                  return;
                }
                setInputValue(`${att.label}:`);
                setPendingEdit(att);
                if (att.type === 'input') {
                  tagInputRef.current.focus();
                  setVisible(false);
                }
              }}>
              {att.label}
            </List.Item>
          );
        }}
      />
    );
  };

  return (
    <div ref={refDiv} className="search-box">
      <TagInput
        ref={tagInputRef}
        size="large"
        prefix={<IconSearchStroked />}
        placeholder={placeholder}
        value={filter(tags, (t: any) => {
          return t.label !== pendingEdit?.value;
        })}
        inputValue={inputValue}
        allowDuplicates={false}
        onInputChange={(value: string) => {
          setInputValue(value);
          if (value === '') {
            setPendingEdit(null);
            setVisible(true);
          }
        }}
        onFocus={() => {
          if (pendingEdit) {
            return;
          }
          setVisible(true);
        }}
        onAdd={(values: string[]) => {
          try {
            if (!pendingEdit) {
              return;
            }
            if (pendingEdit.multiple && !isEmpty(pendingEdit.selectedValue)) {
              searchParams.delete(pendingEdit.value);
              pendingEdit.selectedValue.forEach((v: string) => {
                searchParams.append(pendingEdit.value, v);
              });
              setSearchParams(searchParams);
              return;
            }
            const val = values[0];
            const idx = val.indexOf(':');
            if (idx <= 0) {
              return;
            }
            const tagValue = trim(val.substring(idx + 1));
            if (!isEmpty(tagValue)) {
              searchParams.set(pendingEdit.value, tagValue);
              setSearchParams(searchParams);
            } else {
              if (searchParams.has(pendingEdit.value)) {
                searchParams.delete(pendingEdit.value);
                setSearchParams(searchParams);
              }
            }
          } finally {
            setPendingEdit(null);
            setInputValue('');
          }
        }}
        onRemove={(value: any) => {
          if (searchParams.has(value.label)) {
            searchParams.delete(value.label);
            setSearchParams(searchParams);
          }
        }}
        renderTagItem={(value: any, _index: number, onClose: () => void) => {
          const att = find(attributes, { value: value.label });
          const tagValue = buildTagInputValue(att as any, value.value);
          return (
            <Tag
              key={value.label}
              size="large"
              closable
              onClose={onClose}
              onClick={() => {
                setInputValue(tagValue);
                setPendingEdit(att);
                if (att?.type !== 'input') {
                  setVisible(true);
                }
              }}>
              <Text strong size="small">
                {att?.label}:
              </Text>
              <Text size="small">{att?.multiple ? join(value.value, '|') : value.value}</Text>
            </Tag>
          );
        }}
        showClear
      />
      <Popover
        trigger="custom"
        position="bottomLeft"
        closeOnEsc
        clickToHide={false}
        visible={visible}
        rePosKey={rePosKey}
        onClickOutSide={() => {
          if (pendingEdit) {
            setVisible(false);
          }
        }}
        content={renderSuggests()}>
        <div className="suggest-placeholder" style={{ left: `${rePosKey}px` }}></div>
      </Popover>
    </div>
  );
};
