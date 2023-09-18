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
import { Divider, Form, Select, Tag, Typography } from '@douyinfe/semi-ui';
import { DatasourceStore } from '@src/stores';
import React, { useContext } from 'react';
import { filter, includes, isEmpty } from 'lodash-es';
import { DatasourceInstance } from '@src/types';
import { PlatformContext } from '@src/contexts';
import { useNavigate } from 'react-router-dom';
import { RuleItem } from '@douyinfe/semi-ui/lib/es/form';
import Icon from '../common/Icon';
import { MixedDatasource } from '@src/constants';

const { Text } = Typography;

const DatasourceSelect: React.FC<{
  field?: string;
  label?: string;
  noLabel?: boolean;
  multiple?: boolean;
  includeMixed?: boolean;
  categories?: string[];
  labelPosition?: 'top' | 'left' | 'inset';
  rules?: RuleItem[];
  style?: React.CSSProperties;
}> = (props) => {
  const { label, field, multiple, categories, includeMixed, rules, labelPosition, noLabel, style } = props;
  const datasources = DatasourceStore.getDatasources(includeMixed);
  const { theme } = useContext(PlatformContext);
  const navigate = useNavigate();

  const renderSelectedItem = (n: Record<string, any>): any => {
    const ds = DatasourceStore.getDatasource(n.value);
    if (!ds) {
      return null;
    }
    return (
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {n.value === MixedDatasource ? (
          <Icon icon="mixed" style={{ fontSize: 22 }} />
        ) : (
          <img src={`${ds.plugin.getLogo(theme)}`} width={24} />
        )}
        <Text>{ds.setting.name}</Text>
      </div>
    );
  };
  const renderMultipleSelectedItem = (n: Record<string, any>, { onClose }: any): any => {
    const ds = DatasourceStore.getDatasource(n.value);
    if (!ds) {
      return null;
    }
    const content = (
      <Tag avatarShape="square" closable={true} onClose={onClose} size="large">
        {n.value === MixedDatasource ? (
          <Icon icon="mixed" style={{ fontSize: 18 }} />
        ) : (
          <img src={`${ds.plugin.getLogo(theme)}`} width={20} />
        )}
        <Text style={{ marginLeft: 4 }}>{ds.setting.name}</Text>
      </Tag>
    );
    return {
      isRenderInTag: false,
      content,
    };
  };

  return (
    <Form.Select
      field={field || 'datasource'}
      className="linsight-select"
      multiple={multiple}
      labelPosition={labelPosition}
      label={label}
      noLabel={noLabel}
      rules={rules}
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
      renderSelectedItem={multiple ? renderMultipleSelectedItem : renderSelectedItem}>
      {filter(datasources, (ds: DatasourceInstance) => {
        if (isEmpty(categories)) {
          return true;
        }
        return includes(categories, ds.plugin.category);
      }).map((ds: DatasourceInstance) => {
        const setting = ds.setting;
        return (
          <Select.Option key={setting.uid} value={setting.uid} showTick={false}>
            {setting.uid === MixedDatasource ? (
              <Icon icon="mixed" style={{ fontSize: 22 }} />
            ) : (
              <img src={`${ds.plugin.getLogo(theme)}`} width={24} />
            )}
            <div style={{ marginLeft: 8 }}>
              <Text>{setting.name}</Text>
            </div>
          </Select.Option>
        );
      })}
    </Form.Select>
  );
};

export default DatasourceSelect;
