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
import { Button, SideSheet, Typography } from '@douyinfe/semi-ui';
import { IconClose, IconSaveStroked } from '@douyinfe/semi-icons';
import { DatasourceSelectForm, Icon, MetricExplore, Notification } from '@src/components';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import { ChartSrv } from '@src/services';
import { DatasourceStore } from '@src/stores';
import { Chart, DatasourceInstance, PanelSetting } from '@src/types';
import { ApiKit } from '@src/utils';
import React, { useContext, useState } from 'react';
import { createSearchParams, useNavigate } from 'react-router-dom';
import { get } from 'lodash-es';
const { Text } = Typography;

const ChartDetail: React.FC<{ chart: Chart; setVisible: (v: boolean) => void }> = (props) => {
  const { chart, setVisible } = props;
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const navigate = useNavigate();
  const { datasources } = DatasourceStore;
  const getDatasource = () => {
    const datasourceUID = get(panel, 'datasource.uid', get(datasources, '[0].setting.uid'));
    return DatasourceStore.getDatasource(`${datasourceUID}`);
  };
  const [datasource, setDatasource] = useState<DatasourceInstance | null | undefined>(() => {
    return getDatasource();
  });
  const [submitting, setSubmitting] = useState(false);
  return (
    <div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        <div style={{ flex: 1 }}>
          <div>{chart.title}</div>
          <Text type="tertiary">{chart.desc}</Text>
        </div>
        <DatasourceSelectForm
          noLabel
          value={datasource?.setting.uid}
          style={{ width: 200 }}
          onChange={(instance: DatasourceInstance) => {
            modifyPanel({ datasource: { uid: instance.setting.uid } });
            setDatasource(instance);
          }}
        />
        <Button
          icon={<IconSaveStroked />}
          type="primary"
          loading={submitting}
          onClick={async () => {
            chart.config = panel;
            setSubmitting(true);
            try {
              await ChartSrv.updateChart(chart);
              Notification.success('Save chart successfully!');
            } catch (err) {
              console.warn('save chart error', err);
              Notification.error(ApiKit.getErrorMsg(err));
            } finally {
              setSubmitting(false);
            }
          }}>
          Save
        </Button>
        <Button
          icon={<Icon icon="explore" />}
          type="tertiary"
          onClick={() => {
            const params = createSearchParams({ left: JSON.stringify(get(chart, 'config', null)) });
            navigate({ pathname: '/explore', search: params.toString() });
          }}>
          Explore
        </Button>
        <Button icon={<IconClose />} type="tertiary" onClick={() => setVisible(false)} />
      </div>
      {datasource && <MetricExplore datasource={datasource} />}
    </div>
  );
};

const ChartDetailModal: React.FC<{ chart: Chart; visible: boolean; setVisible: (visible: boolean) => void }> = (
  props
) => {
  const { visible, setVisible, chart } = props;
  return (
    <SideSheet
      className="chart-detail"
      size="large"
      closeOnEsc
      motion={false}
      closable={false}
      visible={visible}
      onCancel={() => setVisible(false)}>
      <PanelEditContextProvider initPanel={(chart.config || {}) as PanelSetting}>
        <ChartDetail chart={chart} setVisible={setVisible} />
      </PanelEditContextProvider>
    </SideSheet>
  );
};

export default ChartDetailModal;
