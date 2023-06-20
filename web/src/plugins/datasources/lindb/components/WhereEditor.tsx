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
  useFieldState,
  Dropdown,
  Form,
  Input,
  List,
  Tag,
  useFormApi,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSearchStroked } from '@douyinfe/semi-icons';
import { StatusTip } from '@src/components';
import { useRequest } from '@src/hooks';
import { isEmpty, has, get, set, pick, transform, isArray, indexOf, join } from 'lodash-es';
import React, { CSSProperties, useCallback, useEffect, useState } from 'react';
import { LinDBDatasource } from '../Datasource';
import { ConditionExpr, Operator } from '../types';
import classNames from 'classnames';

const { Text } = Typography;

const TagValueSelect: React.FC<{
  datasource: LinDBDatasource;
  namespace: string;
  metric: string;
  tagKey: string;
  where: object;
}> = (props) => {
  const { datasource, namespace, metric, tagKey, where } = props;
  const {
    result: tagValues,
    error,
    loading,
  } = useRequest(
    ['load_tag_values_for_where', namespace, metric, tagKey],
    async () => {
      return await datasource.getTagValues(namespace, metric, `${tagKey}`);
    },
    { enabled: !isEmpty(metric) && !isEmpty(tagKey) }
  );

  const getSelectedValue = useCallback(() => {
    const selectedValues = get(where, `${tagKey}.value`);
    if (isArray(selectedValues)) {
      return selectedValues;
    }
    return [selectedValues];
  }, [tagKey, where]);

  const [selected, setSelected] = useState(() => {
    return getSelectedValue();
  });

  useEffect(() => {
    setSelected(getSelectedValue());
  }, [getSelectedValue]);

  return (
    <List
      size="small"
      dataSource={tagValues}
      emptyContent={<StatusTip isLoading={loading} error={error} />}
      header={<Input prefix={<IconSearchStroked />} />}
      renderItem={(item) => (
        <List.Item
          className={classNames('tag-item', {
            active: indexOf(selected, item) >= 0,
          })}
          onClick={() => {
            if (indexOf(selected, item) >= 0) {
              // tag value selcted.
              return;
            }
            const newSelected = [...selected];
            newSelected.push(item);
            setSelected(newSelected);

            if (!has(where, tagKey)) {
              set(where, tagKey, { key: tagKey, operator: Operator.Eq, value: item });
            } else {
              // FIXME: add other op
              const value: any = get(where, `${tagKey}.value`);
              if (isArray(value)) {
                value.push(item);
              } else {
                set(where, tagKey, { key: tagKey, operator: Operator.In, value: [value, item] });
              }
            }
          }}>
          {item}
        </List.Item>
      )}
    />
  );
};

const TagKeySelect: React.FC<{
  datasource: LinDBDatasource;
  namespace: string;
  metric: string;
  currentTagKey: string;
  onChangeTagKey: (tagKey: string) => void;
}> = (props) => {
  const { datasource, namespace, metric, currentTagKey, onChangeTagKey } = props;
  const {
    result: tagKeys,
    error,
    loading,
  } = useRequest(
    ['load_tag_keys_for_where', namespace, metric],
    async () => {
      return await datasource.getTagKeys(namespace, metric);
    },
    { enabled: !isEmpty(metric) }
  );
  return (
    <List
      size="small"
      dataSource={tagKeys}
      header={<Input prefix={<IconSearchStroked />} />}
      emptyContent={<StatusTip isLoading={loading} error={error} />}
      renderItem={(item) => (
        <List.Item
          className={classNames('tag-item', { active: currentTagKey === item })}
          onClick={() => onChangeTagKey(item)}>
          {item}
        </List.Item>
      )}
    />
  );
};

const WhereConditonSelect: React.FC<{
  datasource: LinDBDatasource;
  ns?: string;
  metricField?: string;
  namespaceField?: string;
  style?: CSSProperties;
}> = (props) => {
  const { datasource, ns, metricField = 'metric', namespaceField = 'namespace', style } = props;
  const [visible, setVisible] = useState(false);
  const [currentTagKey, setCurrentTagKey] = useState('');
  const { value: metricName } = useFieldState(metricField);
  const { value: namespace } = useFieldState(namespaceField);
  const formApi = useFormApi();
  const [where, setWhere] = useState<Record<string, ConditionExpr>>(() => {
    const result = {};
    const initWhere = formApi.getValue('where');
    if (isEmpty(initWhere)) {
      return result;
    }
    initWhere.forEach((item: ConditionExpr) => set(result, item.key, item));
    return result;
  });

  const setWhereConditions = useCallback(() => {
    const conditions: ConditionExpr[] = [];
    transform(
      where,
      (_result, value: ConditionExpr, _key) => {
        conditions.push(value);
      },
      {}
    );
    formApi.setValue('where', conditions);
    formApi.submitForm();
  }, [formApi, where]);

  useEffect(() => {
    setWhereConditions();
  }, [visible, setWhereConditions]);

  const pickWhereConditions = (values: string[]) => {
    const finalWhere = pick(where, values);
    setWhere(finalWhere);
    setWhereConditions();
  };

  const conditionToString = (condition: ConditionExpr): string => {
    if (condition.operator === Operator.In) {
      return `${condition.key} ${condition.operator} (${join(condition.value, ',')})`;
    }
    return `${condition.key} ${condition.operator} ${condition.value}`;
  };

  const renderCurrentCondition = () => {
    if (isEmpty(currentTagKey)) {
      return null;
    }
    const condition = get(where, currentTagKey, {}) as ConditionExpr;
    return (
      <>
        <Text size="small" type="quaternary" style={{ marginRight: 2 }}>
          Current:
        </Text>
        <Text size="small">{conditionToString(condition)}</Text>
        <Divider style={{ marginBottom: 8, marginTop: 8 }} />
      </>
    );
  };

  return (
    <Dropdown
      trigger="custom"
      visible={visible}
      onEscKeyDown={() => setVisible(false)}
      onClickOutSide={() => setVisible(false)}
      render={
        <div style={{ padding: 8 }}>
          {renderCurrentCondition()}
          <Row gutter={8}>
            <Col
              span={12}
              style={{
                borderRight: '1px solid var(--semi-color-border)',
              }}>
              <TagKeySelect
                currentTagKey={currentTagKey}
                datasource={datasource}
                namespace={ns || namespace}
                metric={metricName}
                onChangeTagKey={(tagKey: string) => {
                  setCurrentTagKey(tagKey);
                }}
              />
            </Col>
            <Col span={12}>
              <TagValueSelect
                datasource={datasource}
                namespace={ns || namespace}
                metric={metricName}
                tagKey={currentTagKey}
                where={where}
              />
            </Col>
          </Row>
          <div style={{ margin: 8, minWidth: 280 }}>
            <Divider style={{ marginBottom: 8 }} />
            <Row>
              <Col span={12}>
                <Text size="small" type="quaternary" style={{ marginRight: 2 }}>
                  wildcard:
                </Text>
                <Text size="small">host:inst*</Text>
              </Col>
              <Col span={12}>
                <Text size="small" type="quaternary" style={{ marginRight: 2 }}>
                  exclusion:
                </Text>
                <Text size="small">not host:a</Text>
              </Col>
            </Row>
            <Row>
              <Col span={12}>
                <Text size="small" type="quaternary" style={{ marginRight: 2 }}>
                  in:
                </Text>
                <Text size="small">host in (a,b)</Text>
              </Col>
              <Col span={12}>
                <Text size="small" type="quaternary" style={{ marginRight: 2 }}>
                  union:
                </Text>
                <Text size="small">host:a OR host:b</Text>
              </Col>
            </Row>
          </div>
        </div>
      }>
      <Form.TagInput
        field="where"
        labelPosition="inset"
        placeholder="Input where conditions"
        style={style}
        onFocus={() => setVisible(true)}
        showClear
        onChange={(values: string[]) => {
          pickWhereConditions(values);
        }}
        renderTagItem={(value: any, index: number, onClose) => {
          if (isEmpty(value.key) || isEmpty(value.operator) || isEmpty(value.value)) {
            return null;
          }
          return (
            <Tag key={index} closable color="white" size="large" onClose={onClose}>
              {conditionToString(value)}
            </Tag>
          );
        }}
      />
    </Dropdown>
  );
};

export default WhereConditonSelect;
