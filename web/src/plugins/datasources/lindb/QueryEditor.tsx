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
import { Divider, Row, Col, Form, Typography } from '@douyinfe/semi-ui';
import { LinSelect } from '@src/components';
import { QueryEditorProps } from '@src/types';
import React, { useEffect, useState } from 'react';
import { get, clone } from 'lodash-es';
import { LinDBDatasource } from './DataSource';

const { Text } = Typography;

const QueryEditor: React.FC<QueryEditorProps> = (props) => {
  const { datasource, onChange } = props;
  const [params, setParmas] = useState({});
  // convert LinDBDatasource type
  const api = datasource.api as LinDBDatasource;

  const metric = get(params, 'metric', '');
  console.log('change metric....', metric);

  useEffect(() => {
    if (onChange) {
      onChange(params);
    }
  }, [onChange, params]);

  return (
    <Form
      layout="horizontal"
      onValueChange={(values: any) => {
        // FIXME: clone just triger change state
        setParmas(clone(values));
      }}>
      <LinSelect
        style={{ width: 240 }}
        field="metric"
        placeholder="Please select metric"
        labelPosition="inset"
        label="Metric"
        showClear
        filter
        remote
        loader={async () => {
          const values = await api.fetchMetricNames();
          const optionList: any[] = [];
          console.log('kslfasjdflkdsj', values, optionList);
          (values || []).map((item: any) => {
            optionList.push({ value: item, label: item });
          });
          return optionList;
        }}
      />
      <LinSelect
        style={{ width: 240 }}
        field="fields"
        placeholder="Please select fields"
        labelPosition="inset"
        label="Fields"
        showClear
        filter
        remote
        reloadKeys={[metric]}
        loader={async () => {
          const values = await api.getFields(metric);
          const optionList: any[] = [];
          (values || []).map((item: any) => {
            optionList.push({ value: item.name, label: `${item.name}(${item.type})` });
          });
          return optionList;
        }}
      />
      <Form.Select
        field="where"
        label="Where"
        labelPosition="inset"
        style={{ width: 270 }}
        outerBottomSlot={
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
        }
      />
      <Form.Select field="groupBy" label="Group By" labelPosition="inset" />
    </Form>
  );
};

export default QueryEditor;
