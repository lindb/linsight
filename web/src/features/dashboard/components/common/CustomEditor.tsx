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
import { Form, Tag, useFieldState, useFormApi } from '@douyinfe/semi-ui';
import { isEmpty } from 'lodash-es';
import React, { useEffect, useState } from 'react';

const CustomEditor: React.FC = () => {
  const formApi = useFormApi();
  const { value: query } = useFieldState('query');
  const [options, setOptions] = useState<any[]>([]);

  useEffect(() => {
    const parseCustomValue = () => {
      const val = formApi.getValue('query');
      if (isEmpty(val)) {
        return;
      }
      const options: any[] = [];
      const arr = val.split(',');

      arr.forEach((item: string) => {
        if (item.includes(':')) {
          const [key, value] = item.split(':').map((s) => s.trim());
          options.push({ text: key, value: value });
        } else {
          const v = item.trim();
          options.push({ text: v, value: v });
        }
      });
      formApi.setValue('options', options);
      setOptions(options);
    };

    parseCustomValue();
  }, [query, formApi]);

  return (
    <>
      <Form.Slot label="Custom options">
        <Form.TextArea field="query" extraText="Values separated by comma" noLabel />
      </Form.Slot>
      <div style={{ display: 'flex', gap: 2, marginTop: 6 }}>
        {(options || []).map((opt: any, index: number) => (
          <Tag key={index} size="large">
            {opt.text}
          </Tag>
        ))}
      </div>
    </>
  );
};

export default CustomEditor;
