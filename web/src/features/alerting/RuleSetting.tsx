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
import { Button, Card, Form, Steps } from '@douyinfe/semi-ui';
import { IconChevronDown, IconSaveStroked } from '@douyinfe/semi-icons';
import React, { useContext, useEffect, useRef } from 'react';
import { Loading, Panel, QueryEditor } from '@src/components';
import { DatasourceStore } from '@src/stores';
import { MixedDatasource } from '@src/constants';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import { get, isEmpty } from 'lodash-es';
import { AlertingSrv } from '@src/services';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useRequest } from '@src/hooks';
const { Step } = Steps;

const Preview: React.FC = () => {
  const { panel } = useContext(PanelEditContext);
  return (
    <div style={{ height: 280, margin: 8 }}>
      <Panel panel={panel} />
    </div>
  );
};

const AlertData: React.FC = () => {
  return <QueryEditor disableOptions datasource={DatasourceStore.getDatasource(MixedDatasource) as any} />;
};

const Condition: React.FC<{ initValues: object; onChange: (values: any) => void }> = (props) => {
  const { initValues, onChange } = props;
  const formApi = useRef<any>();
  useEffect(() => {
    formApi.current.setValues(initValues);
  }, [initValues]);
  return (
    <Form
      getFormApi={(api: any) => (formApi.current = api)}
      layout="horizontal"
      labelPosition="inset"
      onValueChange={(values: any) => {
        onChange(values);
      }}>
      <Form.InputNumber label="Severity" field="severity" />
      <Form.Input label="Expr" field="expr" />
    </Form>
  );
};

const Setting: React.FC<{ rule?: any }> = (props) => {
  const navigate = useNavigate();
  const { rule } = props;
  const { panel } = useContext(PanelEditContext);
  const condition = useRef<any>();
  const notifications = useRef<any>();
  return (
    <>
      <Preview />
      <Card style={{ margin: 8 }}>
        <Steps className="linsight-steps" type="basic" direction="vertical">
          <Step
            title={
              <div>
                <Button theme="borderless" icon={<IconChevronDown />}>
                  Define metric
                </Button>
              </div>
            }
            description={<AlertData />}
          />
          <Step
            title="Set alert condition"
            status="finish"
            description={
              <Condition
                initValues={get(rule, 'conditions[0]', {})}
                onChange={(values: any) => {
                  condition.current = values;
                }}
              />
            }
          />
          <Step
            title="Notify your team"
            status="process"
            description={
              <Form
                onValueChange={(values: any) => {
                  notifications.current = values.notifications;
                }}>
                <Form.TagInput field="notifications" />
              </Form>
            }
          />
        </Steps>
        <div style={{ display: 'flex', gap: 4 }}>
          <Button
            icon={<IconSaveStroked />}
            onClick={() => {
              const ruleUID = get(rule, 'uid', '');
              const ruleCfg = {
                title: 'Memory > 90%',
                data: {
                  queries: get(panel, 'targets', []),
                },
                conditions: [condition.current],
                notifications: notifications.current,
              } as any;
              if (isEmpty(ruleUID)) {
                AlertingSrv.createAlertRule(ruleCfg);
              } else {
                ruleCfg.uid = ruleUID;
                AlertingSrv.updateAlertRule(ruleCfg);
              }
            }}>
            Save
          </Button>
          <Button
            type="tertiary"
            onClick={() => {
              navigate({
                pathname: '/alerting/rules',
              });
            }}>
            Cancel
          </Button>
        </div>
      </Card>
    </>
  );
};

export const RuleSetting: React.FC = () => {
  console.error('consg.........');
  const defaultDatasource = DatasourceStore.getDefaultDatasource();
  const [searchParams] = useSearchParams();
  const uid = `${searchParams.get('uid')}`;
  const { loading, result: rule } = useRequest(
    ['load-alert-rule', uid],
    async () => {
      return AlertingSrv.getAlertRule(uid);
    },
    { enable: !isEmpty(uid) }
  );
  if (!isEmpty(uid) && loading) {
    return (
      <div className="loading">
        <Loading />
      </div>
    );
  }
  return (
    <PanelEditContextProvider
      initPanel={{
        type: 'timeseries',
        datasource: { uid: MixedDatasource },
        targets: get(rule, 'data.queries') || [
          {
            datasource: { uid: get(defaultDatasource, 'setting.uid', ''), type: defaultDatasource?.plugin.Type },
            request: {},
          },
        ],
      }}>
      <Setting rule={rule} />
    </PanelEditContextProvider>
  );
};
