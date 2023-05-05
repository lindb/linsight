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
import * as _ from 'lodash-es';
import { Button, Select, Card, Typography, Form, Space, useFormApi } from '@douyinfe/semi-ui';
import { IconSaveStroked, IconDeleteStroked } from '@douyinfe/semi-icons';
import { DatasourceSrv } from '@src/services';
import { useQuery } from '@tanstack/react-query';
import { DatasourcePlugin, DatasourceRepositoryInst, DatasourceSetting } from '@src/types';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Notification } from '@src/components';
import { ApiKit } from '@src/utils';

const { Text } = Typography;

const DatasourceSettingForm: React.FC<{ datasource?: DatasourceSetting; SettingEditor: React.ComponentType }> = (
  props
) => {
  const { SettingEditor, datasource } = props;
  const formApi = useFormApi();
  useEffect(() => {
    if (datasource) {
      formApi.setValues(datasource);
    }
  }, [datasource, formApi]);
  return <SettingEditor />;
};

const EditDataSource: React.FC = () => {
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const [type, setType] = useState<string | undefined>(undefined);
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const uid = searchParams.get('uid');
  const plugins = DatasourceRepositoryInst.getPlugins();

  const { data: datasource, isLoading } = useQuery(['load-datasource'], () => DatasourceSrv.getDatasource(`${uid}`), {
    enabled: !_.isEmpty(uid),
  });

  useEffect(() => {
    if (datasource) {
      setType(datasource.type);
    }
  }, [datasource]);

  const PluginSetting = () => {
    const plugin = DatasourceRepositoryInst.get(`${type}`);
    if (!plugin) {
      return null;
    }
    const SettingEditor = plugin?.components?.SettingEditor;
    if (SettingEditor) {
      return <DatasourceSettingForm datasource={datasource} SettingEditor={SettingEditor} />;
    }
    return null;
  };

  return (
    <Card className="linsight-feature" loading={!_.isEmpty(uid) && isLoading}>
      <Form
        labelPosition="left"
        labelAlign="right"
        getFormApi={(api: any) => (formApi.current = api)}
        labelWidth={150}
        disabled={submitting}
        onSubmit={async (values: any) => {
          try {
            setSubmitting(true);
            if (uid) {
              values.uid = uid;
              await DatasourceSrv.updateDatasource(values);
            } else {
              await DatasourceSrv.createDatasource(values);
            }
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Input label="Name" field="name" rules={[{ required: true, message: 'Name is required' }]} />
        <Form.Select
          label="Type"
          field="type"
          disabled={!_.isEmpty(uid)}
          style={{ width: '100%' }}
          onChange={(value: any): void => setType(value)}
          renderSelectedItem={(n: Record<string, any>) => {
            const plugin = DatasourceRepositoryInst.get(`${n.value}`);
            if (!plugin) {
              return null;
            }
            return (
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <img src={`${plugin.darkLogo}`} width={20} />
                <Text>{plugin.Name}</Text>
              </div>
            );
          }}>
          {plugins.map((plugin: DatasourcePlugin) => {
            return (
              <Select.Option key={plugin.Type} value={plugin.Type} showTick={false}>
                <img src={`${plugin.lightLogo}`} width={32} />
                <div style={{ marginLeft: 8 }}>
                  <div>
                    <Text>{plugin.Name}</Text>
                  </div>
                  <Text size="small">{plugin.Description}</Text>
                </div>
              </Select.Option>
            );
          })}
        </Form.Select>

        <PluginSetting />

        <Form.Slot>
          <Space>
            <Button type="tertiary" onClick={() => navigate('/setting/datasources')}>
              Back
            </Button>
            <Button type="tertiary" onClick={() => navigate('/explore')}>
              Explore
            </Button>
            <Button
              type="danger"
              icon={<IconDeleteStroked />}
              onClick={async () => {
                if (uid) {
                  try {
                    await DatasourceSrv.deleteDatasource(uid);
                  } catch (err) {
                    Notification.error(ApiKit.getErrorMsg(err));
                  }
                }
              }}>
              Delete
            </Button>
            <Button icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
              Save & Test
            </Button>
          </Space>
        </Form.Slot>
      </Form>
    </Card>
  );
};

export default EditDataSource;
