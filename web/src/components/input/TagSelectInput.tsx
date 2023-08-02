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
import React, { useState } from 'react';
import { Checkbox, CheckboxGroup, Form, Input, List, Popover, Tag, useFieldState, useFormApi } from '@douyinfe/semi-ui';
import { IconSearchStroked } from '@douyinfe/semi-icons';
import { ColorKit, StringKit } from '@src/utils';
import './tag-select-input.scss';
import { useRequest } from '@src/hooks';
import { TagSrv } from '@src/services';
import { debounce } from 'lodash-es';

const TagSelect: React.FC<{
  values?: string[];
  onChange: (values: any[]) => void;
}> = (props) => {
  const { values, onChange } = props;
  const [searchInput, setSearchInput] = useState('');
  const { result, loading } = useRequest(['load_tags_values', searchInput], () => {
    return TagSrv.findTags(searchInput);
  });

  const search = debounce(setSearchInput, 200);

  return (
    <div>
      <div key="search" style={{ padding: 8 }}>
        <Input key="search" prefix={<IconSearchStroked />} onChange={(v: string) => search(v)} />
      </div>
      <CheckboxGroup
        key="list"
        defaultValue={values}
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

const TagSelectInput: React.FC<{ field?: string; label?: string }> = (props) => {
  const { field = 'tag', label = 'Tag' } = props;
  const [visible, setVisible] = useState(false);
  const { value } = useFieldState(field);
  const formApi = useFormApi();
  return (
    <div className="tag-select-input">
      <Form.TagInput
        label={label}
        field={field}
        allowDuplicates={false}
        onFocus={() => {
          setVisible(true);
        }}
        renderTagItem={(value: string, _index: number, onClose: () => void) => {
          return (
            <Tag
              key={value}
              closable
              onClose={onClose}
              style={{ backgroundColor: ColorKit.getColor(StringKit.hashcode(value)), cursor: 'pointer' }}
              type="solid">
              {value}
            </Tag>
          );
        }}
        showClear
      />
      <Popover
        content={
          <TagSelect
            values={value}
            onChange={(v: string[]) => {
              formApi.setValue(field, v);
            }}
          />
        }
        visible={visible}
        closeOnEsc
        onClickOutSide={() => {
          setVisible(false);
        }}
        trigger="custom"
        position="bottomLeft">
        <div className="suggest-placeholder"></div>
      </Popover>
    </div>
  );
};

export default TagSelectInput;
