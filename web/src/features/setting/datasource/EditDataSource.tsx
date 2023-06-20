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
import React, { useContext, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { isEmpty, get } from 'lodash-es';
import { Button, Select, Card, Typography, Form, Space, Tag } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { DatasourceSrv } from '@src/services';
import { DatasourcePlugin, DatasourceRepositoryInst, DatasourceSetting } from '@src/types';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Icon, Notification } from '@src/components';
import { ApiKit, ObjectKit } from '@src/utils';
import { useRequest } from '@src/hooks';
import { DatasourceStore } from '@src/stores';
import DeleteDatasourceButton from './components/DeleteDatasourceButton';
import { PlatformContext } from '@src/contexts';
import './datasource.scss';

const { Text, Title } = Typography;

const DatasourceSettingForm: React.FC<{
  formApi: any;
  datasource?: DatasourceSetting;
  SettingEditor: React.ComponentType;
}> = (props) => {
  const { SettingEditor, datasource, formApi } = props;
  useEffect(() => {
    if (datasource) {
      formApi.setValues(datasource);
    }
  }, [datasource, formApi]);
  return <SettingEditor />;
};

const EditDataSource: React.FC = () => {
  const { theme, boot } = useContext(PlatformContext);
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);
  const [type, setType] = useState<string | undefined>(undefined);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const uid = searchParams.get('uid');
  const plugins = DatasourceRepositoryInst.getPlugins();

  const { result: datasource, loading } = useRequest(
    ['load-datasource', uid],
    () => DatasourceSrv.getDatasource(`${uid}`),
    {
      enabled: !isEmpty(uid),
    }
  );

  useLayoutEffect(() => {
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
      return <DatasourceSettingForm formApi={formApi.current} datasource={datasource} SettingEditor={SettingEditor} />;
    }
    return null;
  };
  const gotoDatasourceList = () => {
    navigate({ pathname: '/setting/datasources' });
  };

  return (
    <Card
      loading={!isEmpty(uid) && loading}
      className="setting-page"
      bordered={false}
      bodyStyle={{ padding: 12 }}
      title={
        <Card.Meta
          className="setting-meta"
          title={
            <div className="meta-title">
              <Title heading={3} style={{ cursor: 'pointer' }} onClick={gotoDatasourceList} underline>
                Datasource
              </Title>
              <Title heading={3}>/ {isEmpty(uid) ? 'New Datasource' : get(datasource, 'name', 'N/A')}</Title>
            </div>
          }
          description={
            <div style={{ display: 'flex', gap: 8 }}>
              <Text>Current organization:</Text>
              <Tag>{get(boot, 'user.org.name', 'N/A')}</Tag>
            </div>
          }
          avatar={<Icon icon="datasource" />}
        />
      }>
      <Form
        className="linsight-form datasource-form"
        labelPosition="left"
        labelAlign="right"
        getFormApi={(api: any) => (formApi.current = api)}
        labelWidth={150}
        allowEmpty
        disabled={submitting}
        onSubmit={async (values: any) => {
          try {
            setSubmitting(true);
            if (uid) {
              values.uid = uid;
              ObjectKit.merge(datasource, values);
              await DatasourceSrv.updateDatasource(values);
            } else {
              const uid = await DatasourceSrv.createDatasource(values);
              searchParams.set('uid', `${uid}`);
              setSearchParams(searchParams);
            }
            DatasourceStore.syncDatasources();
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
          disabled={!isEmpty(uid)}
          style={{ width: '100%' }}
          onChange={(value: any): void => setType(value)}
          renderSelectedItem={(n: Record<string, any>) => {
            const plugin = DatasourceRepositoryInst.get(`${n.value}`);
            if (!plugin) {
              return null;
            }
            return (
              <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                <img src={`${plugin.getLogo(theme)}`} width={20} />
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
            <Button icon={<Icon icon="explore" />} type="tertiary" onClick={() => navigate('/explore')}>
              Explore
            </Button>
            {uid && (
              <DeleteDatasourceButton
                uid={uid}
                name={datasource?.name}
                onCompleted={() => {
                  navigate({
                    pathname: '/setting/datasources',
                  });
                }}
                text="Delete"
              />
            )}
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
