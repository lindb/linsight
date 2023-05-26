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
import React from 'react';
import { Button, Form, Radio } from '@douyinfe/semi-ui';
import { IconTick, IconDeleteStroked } from '@douyinfe/semi-icons';
import { DatasourceRepositoryInst, VariableHideType, VariableType, OptionType } from '@src/types';
import { DatasourceSelect } from '@src/components';
import { useSearchParams } from 'react-router-dom';
import { DashboardStore, DatasourceStore } from '@src/stores';
import { observer } from 'mobx-react-lite';
import { get } from 'lodash-es';

const VariableEditor: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const index = `${searchParams.get('edit')}`;
  const variable = DashboardStore.getVariableByIndex(index);

  const gotoList = () => {
    searchParams.delete('edit');
    setSearchParams(searchParams);
  };
  /*
   * Render datasource plugin variable eidtor if datasource implements.
   */
  const QueryEditor = (props: { formState: any }) => {
    const { formState } = props;
    const datasource = DatasourceStore.getDatasource(get(formState, 'values.query.datasource.uid', ''));
    console.log('xxxx....', datasource, formState);
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
    return <VariableQueryEditor variable={formState.values} datasource={datasource} />;
  };

  /*
   * Query variable editor
   */
  const QueryVaribleEditor = (props: { formState: any }) => {
    const { formState } = props;
    if (get(formState, 'values.type') !== VariableType.Query) {
      return null;
    }
    return (
      <>
        <DatasourceSelect
          rules={[{ required: true, message: 'Datasource is required.' }]}
          label="Data source"
          style={{ width: 300 }}
          field="query.datasource.uid"
        />
        <QueryEditor formState={formState} />
      </>
    );
  };

  return (
    <Form
      className="linsight-form"
      labelPosition="top"
      extraTextPosition="middle"
      initValues={variable}
      onSubmit={(values: any) => {
        console.log('apply submit', values);
        DashboardStore.updateVariable(index, values);
        gotoList();
      }}>
      {({ formState }) => (
        <>
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
              { value: VariableType.Query, label: 'Query' },
              { value: VariableType.Constant, label: 'Constant' },
            ]}
          />
          <QueryVaribleEditor formState={formState} />
          <Form.Slot label="Selection options">
            <Form.Checkbox field="multi" value={OptionType.Multi} noLabel>
              Multi-value
            </Form.Checkbox>
            <Form.Checkbox field="includeAll" value={OptionType.All} noLabel>
              Include all option
            </Form.Checkbox>
          </Form.Slot>
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
        </>
      )}
    </Form>
  );
};

export default observer(VariableEditor);
