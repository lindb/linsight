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
import { Divider, Form, Select, Typography } from '@douyinfe/semi-ui';
import { DatasourceStore } from '@src/stores';
import React, { MutableRefObject, useContext, useRef } from 'react';
import { isEmpty } from 'lodash-es';
import { DatasourceInstance } from '@src/types';
import { PlatformContext } from '@src/contexts';
import { useNavigate } from 'react-router-dom';

const { Text } = Typography;

const DatasourceSelect: React.FC<{
  value?: string | null;
  label?: string;
  noLabel?: boolean;
  labelPosition?: 'top' | 'left' | 'inset';
  style?: React.CSSProperties;
  onChange?: (instance: DatasourceInstance) => void;
}> = (props) => {
  const { value, label, labelPosition, noLabel, style, onChange } = props;
  const { datasources } = DatasourceStore;
  const { theme } = useContext(PlatformContext);
  const previousValue = useRef() as MutableRefObject<string>;
  const navigate = useNavigate();

  return (
    <Form
      initValues={{ datasource: value }}
      onValueChange={(values: any) => {
        if (!onChange) {
          return;
        }
        const value = values['datasource'];
        if (value === previousValue.current) {
          return;
        }
        previousValue.current = value;
        const ds = DatasourceStore.getDatasource(value);
        if (ds) {
          onChange(ds);
        }
      }}>
      <Form.Select
        field="datasource"
        className="linsight-select"
        labelPosition={labelPosition}
        label={label}
        noLabel={noLabel}
        style={style}
        outerBottomSlot={
          // if no data source, add new datasource link
          isEmpty(datasources) && (
            <div style={{ textAlign: 'center', height: 36 }}>
              <Divider style={{ marginBottom: 8 }} />
              <Text link onClick={() => navigate('/setting/datasource')}>
                Create a datasource
              </Text>
            </div>
          )
        }
        renderSelectedItem={(n: Record<string, any>) => {
          const ds = DatasourceStore.getDatasource(n.value);
          if (!ds) {
            return null;
          }
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <img src={`${ds.plugin.getLogo(theme)}`} width={24} />
              <Text>{ds.setting.name}</Text>
            </div>
          );
        }}>
        {datasources.map((ds: DatasourceInstance) => {
          const setting = ds.setting;
          return (
            <Select.Option key={setting.uid} value={setting.uid} showTick={false}>
              <img src={`${ds.plugin.getLogo(theme)}`} width={24} />
              <div style={{ marginLeft: 8 }}>
                <Text>{setting.name}</Text>
              </div>
            </Select.Option>
          );
        })}
      </Form.Select>
    </Form>
  );
};

export default DatasourceSelect;

