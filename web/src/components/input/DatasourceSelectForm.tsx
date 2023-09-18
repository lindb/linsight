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
import { DatasourceStore } from '@src/stores';
import React, { MutableRefObject, useEffect, useMemo, useRef } from 'react';
import { DatasourceInstance, Tracker } from '@src/types';
import { DatasourceSelect } from '@src/components';

const DatasourceSelectForm: React.FC<{
  value?: string;
  label?: string;
  noLabel?: boolean;
  categories?: string[];
  labelPosition?: 'top' | 'left' | 'inset';
  includeMixed?: boolean;
  style?: React.CSSProperties;
  onChange?: (instance: DatasourceInstance) => void;
}> = (props) => {
  const { value, label, categories, labelPosition, noLabel, includeMixed, style, onChange } = props;
  const tracker = useRef() as MutableRefObject<Tracker<string | undefined>>;
  const formApi = useRef<any>();

  useMemo(() => {
    tracker.current = new Tracker(value);
  }, [value]); // NOTE: just init

  useEffect(() => {
    if (formApi.current) {
      formApi.current.setValues({ datasource: value });
    }
  }, [value]);

  return (
    <Form
      getFormApi={(api: any) => {
        formApi.current = api;
      }}
      onValueChange={(values: any) => {
        if (!onChange) {
          return;
        }

        const value = values['datasource'];

        if (tracker.current.isChanged(value)) {
          tracker.current.setNewVal(value);

          const ds = DatasourceStore.getDatasource(value);
          if (ds) {
            onChange(ds);
          }
        }
      }}>
      <DatasourceSelect
        labelPosition={labelPosition}
        label={label}
        noLabel={noLabel}
        style={style}
        categories={categories}
        includeMixed={includeMixed}
      />
    </Form>
  );
};

export default DatasourceSelectForm;
