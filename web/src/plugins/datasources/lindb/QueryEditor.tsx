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
import { Button, Form, Tooltip, Typography } from '@douyinfe/semi-ui';
import { IconChevronDown, IconChevronRight, IconHelpCircleStroked } from '@douyinfe/semi-icons';
import { Query, QueryEditorProps } from '@src/types';
import React, { MutableRefObject, useContext, useRef, useState } from 'react';
import { get, isEmpty } from 'lodash-es';
import { LinDBDatasource } from './Datasource';
import { QueryEditContext } from '@src/contexts';
import MetricNameSelect from './components/MetricNameSelect';
import TagKeySelect from './components/TagKeySelect';
import FieldSelect from './components/FieldSelect';
import WhereConditonSelect from './components/WhereEditor';
import './query-edit.scss';
import { ObjectKit } from '@src/utils';
import NamespaceSelect from './components/NamespaceSelect';

const { Text } = Typography;

const OptionsContent: React.FC<{ values: object }> = (props) => {
  const { values } = props;
  const legend = get(values, 'legendFormat', 'Auto');
  const info = [];
  info.push(<span>Legend: {isEmpty(legend) ? 'Auto' : legend}</span>);
  info.push(<span>Include field: {`${get(values, 'includeField', false)}`}</span>);
  return <span className="options-item">{info}</span>;
};

const OptionsEditor: React.FC = () => {
  const [collapsed, setCollapsed] = useState(false);
  const formApi = useRef() as MutableRefObject<any>;
  const { target, modifyTarget } = useContext(QueryEditContext);
  const [values, setValues] = useState<object>(target);
  return (
    <div className="options-editor">
      <div className="options-header" onClick={() => setCollapsed(!collapsed)}>
        <Button
          icon={collapsed ? <IconChevronDown /> : <IconChevronRight />}
          size="small"
          theme="borderless"
          type="tertiary"
        />
        <div>Options</div>
        {!collapsed && (
          <Text type="tertiary" size="small">
            <OptionsContent values={values} />
          </Text>
        )}
      </div>
      <div style={{ display: collapsed ? 'block' : 'none' }}>
        <Form
          layout="horizontal"
          allowEmpty
          className="lindb-query-editor options-form"
          getFormApi={(api: any) => (formApi.current = api)}
          initValues={target}
          onSubmit={(values: any) => {
            setValues(values);
            modifyTarget(values);
          }}
          onBlur={() => {
            if (formApi.current) {
              formApi.current.submitForm();
            }
          }}>
          <Form.Input
            field="legendFormat"
            placeholder="Auto"
            label={
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <Text style={{ color: 'var(--semi-color-text-1)' }}>Legend</Text>
                <Tooltip
                  content={
                    <div style={{ display: 'flex', flexDirection: 'column' }}>
                      <div>Series name override/template</div>
                      <ul style={{ padding: '0px 12px', margin: 0 }}>
                        <li>{'{{host}}'} will be replaced with label value for host</li>
                      </ul>
                    </div>
                  }>
                  <IconHelpCircleStroked size="small" />
                </Tooltip>
              </div>
            }
            style={{ width: 180 }}
          />
          <Form.Checkbox label="Include field" field="includeField" onChange={() => formApi.current.submitForm()} />
        </Form>
      </div>
    </div>
  );
};

const QueryEditor: React.FC<QueryEditorProps> = (props) => {
  const { datasource } = props;
  const { target, modifyTarget } = useContext(QueryEditContext);
  const api = datasource.api as LinDBDatasource; // covert LinDB datasource

  return (
    <>
      <Form
        allowEmpty
        className="lindb-query-editor"
        layout="horizontal"
        initValues={get(target, 'request', {})}
        onSubmit={(values: any) => {
          const newValues = ObjectKit.cleanEmptyProperties(values);
          // change query edit context's values
          modifyTarget({ request: newValues } as Query);
        }}>
        {({ formApi }) => (
          <>
            <NamespaceSelect datasource={api} style={{ minWidth: 240 }} />
            <MetricNameSelect datasource={api} style={{ minWidth: 240 }} />
            <FieldSelect datasource={api} style={{ minWidth: 240 }} />
            <WhereConditonSelect datasource={api} style={{ minWidth: 270 }} />
            <TagKeySelect
              field="groupBy"
              placeholder="Please select group by tag key"
              multiple
              label="Group By"
              datasource={api}
              onFinished={() => formApi.submitForm()}
            />
          </>
        )}
      </Form>
      <OptionsEditor />
    </>
  );
};

export default QueryEditor;
