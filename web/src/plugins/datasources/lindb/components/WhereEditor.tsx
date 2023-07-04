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
  useFieldState,
  useFormApi,
  useFormState,
  Button,
  Select,
  Radio,
  Popover,
  Col,
  Row,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlusStroked, IconCrossStroked, IconHelpCircleStroked } from '@douyinfe/semi-icons';
import { useRequest } from '@src/hooks';
import { isEmpty, pick, isArray, find, remove, startsWith, filter, includes, debounce } from 'lodash-es';
import React, { MutableRefObject, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { LinDBDatasource } from '../Datasource';
import { ConditionExpr, Operator } from '../types';
import { Tracker } from '@src/types';
import { VariableContext } from '@src/contexts';
const { Text } = Typography;

const TagValueSelect: React.FC<{
  datasource: LinDBDatasource;
  namespace: string;
  metric: string;
  condition: ConditionExpr;
  conditions: ConditionExpr[];
  onConditionChange: (condition: ConditionExpr) => void;
}> = (props) => {
  const { datasource, namespace, metric, condition, conditions, onConditionChange } = props;
  const { definitions } = useContext(VariableContext);
  const [tagValues, setTagValues] = useState<string[]>([]);
  const [input, setInput] = useState('');
  const fetchTagValues = useCallback(async () => {
    if (startsWith(input, '$')) {
      setTagValues(
        (definitions || []).map((d: any) => {
          return `\${${d.name}}`;
        })
      );
      return;
    }
    const where = filter(conditions, (o: ConditionExpr) => {
      return o.key !== condition.key && !isEmpty(o.value);
    });
    const tagValues = await datasource.getTagValues(namespace, metric, condition.key, where, input);
    setTagValues(tagValues);
  }, [condition, conditions, datasource, definitions, namespace, metric, input]);

  useEffect(() => {
    fetchTagValues();
  }, [input, fetchTagValues]);

  const search = debounce(setInput, 200);

  return (
    <Select
      placeholder="Tag value"
      style={{ minWidth: 100 }}
      multiple={condition.operator === Operator.In}
      filter
      remote
      allowCreate
      showClear
      onFocus={() => {
        fetchTagValues();
      }}
      onChange={(values: any) => {
        condition.value = values;
        onConditionChange(condition);
      }}
      onCreate={(op: any) => {
        op.showTick = false;
      }}
      onSearch={(v: string) => {
        search(v);
      }}
      defaultValue={condition.value}>
      {(tagValues || []).map((tagValue: string) => {
        return (
          <Select.Option value={tagValue} showTick={false} key={tagValue}>
            {tagValue}
          </Select.Option>
        );
      })}
    </Select>
  );
};

const ConditionInput: React.FC<{
  datasource: LinDBDatasource;
  condition: ConditionExpr;
  conditions: ConditionExpr[];
  namespace: string;
  metric: string;
  tagKeys: string[];
  onConditionChange: (condition: ConditionExpr) => void;
  onConditionRemove: (condition: ConditionExpr) => void;
}> = (props) => {
  const { datasource, namespace, tagKeys, metric, condition, conditions, onConditionChange, onConditionRemove } = props;
  return (
    <div style={{ display: 'flex', alignItems: 'center', height: '100%' }}>
      <Select
        placeholder="Tag key"
        defaultValue={condition.key}
        onChange={(v: any) => {
          condition.key = v;
          onConditionChange(condition);
        }}>
        {(tagKeys || []).map((tagKey: string) => {
          return (
            <Select.Option value={tagKey} showTick={false} key={tagKey}>
              {tagKey}
            </Select.Option>
          );
        })}
      </Select>
      <Select
        className="operator"
        showArrow={false}
        defaultValue={condition.operator || Operator.Eq}
        optionList={[
          { label: Operator.Eq, value: Operator.Eq, showTick: false },
          { label: Operator.In, value: Operator.In, showTick: false },
          { label: Operator.Like, value: Operator.Like, showTick: false },
        ]}
        onChange={(v: any) => {
          condition.operator = v;
          if (v === Operator.In) {
            condition.value = [condition.value as string];
          } else if (isArray(condition.value)) {
            condition.value = condition.value[0];
          }
          onConditionChange(condition);
        }}
      />
      <TagValueSelect
        conditions={conditions}
        condition={condition}
        datasource={datasource}
        namespace={namespace}
        metric={metric}
        onConditionChange={onConditionChange}
      />
      <div
        style={{
          backgroundColor: 'var(--semi-color-fill-0)',
          height: '100%',
          display: 'flex',
          alignItems: 'center',
        }}>
        <Popover
          showArrow
          content={
            <div>
              <Text size="small">Optional: </Text>
              <Text size="small" type="tertiary">
                If the tag value is not selected, then ignore this condition
              </Text>
            </div>
          }>
          <Radio
            mode="advanced"
            checked={condition.optional}
            onChange={() => {
              condition.optional = !condition.optional;
              onConditionChange(condition);
            }}
          />
        </Popover>
      </div>
      <Button
        icon={<IconCrossStroked />}
        type="danger"
        onClick={() => {
          onConditionRemove(condition);
        }}
      />
    </div>
  );
};

const WhereConditonSelect: React.FC<{
  datasource: LinDBDatasource;
  ns?: string;
  metricField?: string;
  namespaceField?: string;
}> = (props) => {
  const { datasource, ns, metricField = 'metric', namespaceField = 'namespace' } = props;
  const { value: metricName } = useFieldState(metricField);
  const { value: namespace } = useFieldState(namespaceField);
  const formApi = useFormApi();
  const formState = useFormState();
  const formValues = formState.values;
  const [conditions, setConditions] = useState<ConditionExpr[]>(() => {
    return formApi.getValue('where') || ([{}] as ConditionExpr[]);
  });

  const namespaceAndMetric = pick(formValues, [metricField, namespaceField]);
  const reloadKeysTracker = useRef() as MutableRefObject<Tracker<any>>;
  const [tagKeys, setTagKeys] = useState<string[]>([]);
  const { result } = useRequest(
    ['load_tag_keys_for_where', namespace, metricName],
    async () => {
      return await datasource.getTagKeys(namespace, metricName);
    },
    { enabled: !isEmpty(metricName) }
  );

  /**
   * initialize value
   */
  useMemo(() => {
    reloadKeysTracker.current = new Tracker(namespaceAndMetric);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // just init

  useEffect(() => {
    formApi.setValue(
      'where',
      filter(conditions, (c: ConditionExpr) => {
        return !isEmpty(c.value);
      })
    );
    formApi.submitForm();
  }, [conditions, formApi]);

  useEffect(() => {
    const selectedKey = (conditions || []).map((c: ConditionExpr) => c.key);
    setTagKeys(filter(result as string[], (key: string) => !includes(selectedKey, key)));
  }, [result, conditions]);

  useEffect(() => {
    if (reloadKeysTracker.current.isChanged(namespaceAndMetric)) {
      reloadKeysTracker.current.setNewVal(namespaceAndMetric);
      setConditions([{} as ConditionExpr]);
      formApi.setValue('where', []);
    }
  }, [namespaceAndMetric, formApi]);

  return (
    <div style={{ display: 'inline-flex', width: '100%' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <div
          className="semi-select-inset-label"
          style={{
            display: 'flex',
            alignItems: 'center',
            padding: '0px 12px',
            marginRight: 0,
            gap: 2,
            height: '100%',
            backgroundColor: 'var(--semi-color-fill-0)',
          }}>
          <span>Where</span>
          <Popover
            showArrow
            content={
              <div style={{ width: 300 }}>
                <Row>
                  <Col span={12}>
                    <Text size="small" type="tertiary" style={{ marginRight: 2 }}>
                      wildcard:
                    </Text>
                    <Text size="small">host:inst*</Text>
                  </Col>
                  <Col span={12}>
                    <Text size="small" type="tertiary" style={{ marginRight: 2 }}>
                      exclusion:
                    </Text>
                    <Text size="small">not host:a</Text>
                  </Col>
                </Row>
                <Row>
                  <Col span={12}>
                    <Text size="small" type="tertiary" style={{ marginRight: 2 }}>
                      in:
                    </Text>
                    <Text size="small">host in (a,b)</Text>
                  </Col>
                  <Col span={12}>
                    <Text size="small" type="tertiary" style={{ marginRight: 2 }}>
                      union:
                    </Text>
                    <Text size="small">host:a OR host:b</Text>
                  </Col>
                </Row>
              </div>
            }>
            <IconHelpCircleStroked style={{ cursor: 'pointer' }} />
          </Popover>
        </div>
        {conditions.map((condition: ConditionExpr, index: number) => (
          <ConditionInput
            key={`${condition.key}-${index}`}
            tagKeys={tagKeys}
            condition={condition}
            conditions={conditions}
            datasource={datasource}
            namespace={ns || namespace}
            metric={metricName}
            onConditionChange={(condition: ConditionExpr) => {
              // set default op
              condition.operator = condition.operator || Operator.Eq;
              setConditions([...conditions]);
            }}
            onConditionRemove={(condition: ConditionExpr) => {
              remove(conditions, (expr: ConditionExpr) => {
                return condition.key === expr.key;
              });
              setConditions([...conditions]);
            }}
          />
        ))}
        <Button
          icon={<IconPlusStroked />}
          type="tertiary"
          onClick={() => {
            if (
              !isEmpty(tagKeys) &&
              !find(conditions, (c: ConditionExpr) => {
                return isEmpty(c.key);
              })
            ) {
              conditions.push({} as ConditionExpr);
              setConditions([...conditions]);
            }
          }}
        />
      </div>
    </div>
  );
};

export default WhereConditonSelect;
