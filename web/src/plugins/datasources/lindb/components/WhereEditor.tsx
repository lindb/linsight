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
import {
  Divider,
  Row,
  Col,
  Typography,
  useFieldState,
  Transfer,
  Dropdown,
  Form,
  Input,
  List,
  Tag,
  useFormApi,
} from '@douyinfe/semi-ui';
import { IconSearchStroked, IconPlusStroked, IconPlusCircleStroked } from '@douyinfe/semi-icons';
import { LinSelect } from '@src/components';
import { useRequest } from '@src/hooks';
import { DatasourceInstance } from '@src/types';
import { isEmpty } from 'lodash-es';
import React, { CSSProperties, useEffect, useState } from 'react';
import { LinDBDatasource } from '../Datasource';
import { ConditionExpr, Operator } from '../types';

const { Text } = Typography;

const TagValueSelect: React.FC<{
  datasource: LinDBDatasource;
  metric: string;
  tagKey: string | undefined;
}> = (props) => {
  const { datasource, metric, tagKey } = props;
  const {
    result: tagValues,
    error,
    loading,
    refetch,
  } = useRequest(
    ['load_tag_values_for_where', metric, tagKey],
    async () => {
      return await datasource.getTagValues(metric, `${tagKey}`);
    },
    { enabled: !isEmpty(metric) && !isEmpty(tagKey) }
  );
  console.log('tag values load', tagValues);
  return (
    <List
      dataSource={tagValues}
      header={<Input prefix={<IconSearchStroked />} />}
      renderItem={(item) => <List.Item>{item}</List.Item>}
      size="small"
    />
  );
};

const TagKeySelect: React.FC<{
  datasource: LinDBDatasource;
  metric: string;
  onChangeTagKey: (tagKey: string) => void;
}> = (props) => {
  const { datasource, metric, onChangeTagKey } = props;
  const {
    result: tagKeys,
    error,
    loading,
    refetch,
  } = useRequest(
    ['load_tag_keys_for_where', metric],
    async () => {
      return await datasource.getTagKeys(metric);
    },
    { enabled: !isEmpty(metric) }
  );
  return (
    <List
      dataSource={tagKeys}
      header={<Input prefix={<IconSearchStroked />} />}
      renderItem={(item) => (
        <List.Item
          main={<span onClick={() => onChangeTagKey(item)}>{item}</span>}
          extra={
            <IconPlusCircleStroked
              style={{ cursor: 'pointer', color: 'var(--semi-color-success)' }}
              onClick={() => {
                console.log('xxxxxxxxxxxx');
              }}
            />
          }
        />
      )}
      size="small"
    />
  );
};

const WhereConditonSelect: React.FC<{
  datasource: LinDBDatasource;
  metricField?: string;
  style?: CSSProperties;
}> = (props) => {
  const { datasource, metricField = 'metric', style } = props;
  const [visible, setVisible] = useState(false);
  const [currentTagKey, setCurrentTagKey] = useState('');
  const { value: metricName } = useFieldState(metricField);
  const where: ConditionExpr[] = [{ key: 'node', operator: Operator.Eq, value: '${node}' }];
  const formApi = useFormApi();
  useEffect(() => {
    // FIXME: mock need remove
  }, []);
  useEffect(() => {
    if (visible) {
      formApi.setValue('where', where);
      formApi.submitForm();
    }
  }, [visible]);

  return (
    <Dropdown
      trigger="custom"
      visible={visible}
      onEscKeyDown={() => setVisible(false)}
      onClickOutSide={() => setVisible(false)}
      render={
        <div>
          <Row>
            <Col span={12}>
              <TagKeySelect
                datasource={datasource}
                metric={metricName}
                onChangeTagKey={(tagKey: string) => setCurrentTagKey(tagKey)}
              />
            </Col>
            <Col span={12}>
              <TagValueSelect datasource={datasource} metric={metricName} tagKey={currentTagKey} />
            </Col>
          </Row>
        </div>
      }>
      <Form.TagInput
        field="where"
        labelPosition="inset"
        placeholder="Input where conditions"
        style={style}
        onFocus={() => setVisible(true)}
        showClear
        onChange={(values: any) => {
          console.log('change......', values);
        }}
        onRemove={(removedValue: string, idx: number) => {
          // formApi.setValue('where', null);
          console.log('on remove xxxx..........', formApi.getValues());
        }}
        renderTagItem={(value: any, index: number) => (
          <Tag
            key={index}
            closable
            color="white"
            size="large"
            onClose={() => {
              console.log('xxxx...........');
              // formApi.setValue('where', null);
              console.log('xxxx..........', formApi.getValues());
              // formApi.submitForm();
            }}>
            {value.key + value.operator + value.value}
          </Tag>
        )}
      />
    </Dropdown>
  );
};

export default WhereConditonSelect;
