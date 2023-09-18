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
import { Form } from '@douyinfe/semi-ui';
import React, { useEffect, useRef } from 'react';

const DynamicForm: React.FC<{
  layout?: 'horizontal' | 'vertical';
  fields: any[];
  initValues?: object;
  onValueChange?: (value: any) => void;
}> = (props) => {
  const { fields = [], initValues, layout, onValueChange = (_values: any) => {} } = props;
  const formApi = useRef<any>();

  useEffect(() => {
    formApi.current.setValues(initValues);
  }, [initValues]);

  return (
    <Form
      getFormApi={(api: any) => {
        formApi.current = api;
      }}
      layout={layout}
      className="linsight-form"
      labelPosition="left"
      onValueChange={(values: any) => {
        onValueChange(values);
      }}>
      {(fields || []).map((f: any, idx: number) => {
        switch (f.type) {
          case 'Input':
            return <Form.Input key={idx} {...f.props} />;
          case 'TextArea':
            return <Form.TextArea key={idx} {...f.props} />;
        }
      })}
    </Form>
  );
};

export default DynamicForm;
