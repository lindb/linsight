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
import React, { useEffect, useRef, useState } from 'react';
import { Button, Form, Radio } from '@douyinfe/semi-ui';
import { IconTick, IconDeleteStroked } from '@douyinfe/semi-icons';
import { DatasourceInstance, DatasourceRepositoryInst, VariableHideType } from '@src/types';
import { DatasourceSelect } from '@src/components';
import { useSearchParams } from 'react-router-dom';
import { DashboardStore } from '@src/stores';
import { observer } from 'mobx-react-lite';

const VariableEditor: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(null);
  const index = `${searchParams.get('edit')}`;
  const variable = DashboardStore.getVariableByIndex(index);
  const formApi = useRef<any>();
  useEffect(() => {
    if (formApi.current) {
      formApi.current.setValues(variable);
    }
  }, [variable, formApi]);

  const QueryEditor = () => {
    if (!datasource) {
      return null;
    }
    const plugin = DatasourceRepositoryInst.get(datasource.plugin.Type);
    if (!plugin) {
      return null;
    }
    const VariableQueryEditor = plugin.components.VariableEditor;
    if (!VariableQueryEditor) {
      return null;
    }
    return <VariableQueryEditor />;
  };
  const gotoList = () => {
    searchParams.delete('edit');
    setSearchParams(searchParams);
  };

  return (
    <Form
      className="linsight-form"
      labelPosition="top"
      extraTextPosition="middle"
      getFormApi={(api: any) => (formApi.current = api)}
      onSubmit={(values: any) => {
        DashboardStore.updateVariable(index, values);
        gotoList();
      }}>
      <Form.Input
        field="name"
        label="Name"
        extraText="The name of variable"
        rules={[{ required: true, message: 'Name is required.' }]}
      />
      <Form.Input field="label" label="Label" extraText="Display name(optional)" />
      <Form.RadioGroup field="hide" label="Show" type="button">
        <Radio value={VariableHideType.LabelAndValue}>Label and value</Radio>
        <Radio value={VariableHideType.OnlyValue}>Value</Radio>
        <Radio value={VariableHideType.Hide}>Nothing</Radio>
      </Form.RadioGroup>
      <Form.Select
        field="type"
        label="Type"
        style={{ width: 300 }}
        rules={[{ required: true, message: 'Type is required.' }]}
        optionList={[
          { value: 'query', label: 'Query' },
          { value: 'constant', label: 'Constant' },
        ]}
      />
      <DatasourceSelect
        label="Data source"
        style={{ width: 300 }}
        onChange={(instance: DatasourceInstance) => {
          setDatasource(instance);
        }}
      />
      <QueryEditor />
      <Form.CheckboxGroup field="Selection options">
        <Form.Checkbox label="Multi-value">Multi-value</Form.Checkbox>
        <Form.Checkbox label="Include all option">Include all option</Form.Checkbox>
      </Form.CheckboxGroup>
      <Form.Slot>
        <div style={{ gap: 4, display: 'flex', flexDirection: 'row' }}>
          <Button
            type="danger"
            icon={<IconDeleteStroked />}
            onClick={() => {
              DashboardStore.deleteVariable(index);
              gotoList();
            }}>
            Delete
          </Button>
          <Button
            type="secondary"
            onClick={() => {
              gotoList();
            }}>
            Back
          </Button>
          <Button type="primary" icon={<IconTick />} htmlType="submit">
            Apply
          </Button>
        </div>
      </Form.Slot>
    </Form>
  );
};

export default observer(VariableEditor);
