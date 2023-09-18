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
import { DatasourceSelectForm, Icon, IntegrationIcon, DataExplore, Notification } from '@src/components';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import { ChartSrv } from '@src/services';
import { DatasourceStore } from '@src/stores';
import { Chart, DatasourceCategory, DatasourceInstance, PanelSetting } from '@src/types';
import { ApiKit } from '@src/utils';
import React, { useContext, useRef, useState } from 'react';
import { createSearchParams, useNavigate } from 'react-router-dom';
import { get, unset } from 'lodash-es';
import { Resizable } from 're-resizable';
const { Text, Title } = Typography;

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
        <IntegrationIcon integration={chart.integration} style={{ fontSize: 32 }} />
        <div style={{ flex: 1 }}>
          <Title heading={5}>{chart.title}</Title>
          <Text type="tertiary">{chart.description || 'N/A'}</Text>
        </div>
        <DatasourceSelectForm
          noLabel
          value={datasource?.setting.uid}
          categories={[DatasourceCategory.Metric]}
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
            setSubmitting(true);
            try {
              await ChartSrv.updateChart({ uid: chart.uid, ...panel });
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
            const panel = get(chart, 'model', {});
            unset(panel, 'title');
            unset(panel, 'description');
            const params = createSearchParams({ left: JSON.stringify(panel) });
            navigate({ pathname: '/explore', search: params.toString() });
          }}>
          Explore
        </Button>
        <Button icon={<IconClose />} type="tertiary" onClick={() => setVisible(false)} />
      </div>
      {datasource && <DataExplore datasource={datasource} />}
    </div>
  );
};

const ChartDetailModal: React.FC<{ chart: Chart; visible: boolean; setVisible: (visible: boolean) => void }> = (
  props
) => {
  const { visible, setVisible, chart } = props;
  const initWidth = useRef(document.body.clientWidth * 0.55);
  const [width, setWidth] = useState(initWidth.current);

  return (
    <SideSheet
      className="chart-detail split-view"
      headerStyle={{ padding: 0 }}
      size="large"
      closeOnEsc
      width={width}
      height={'100vh'}
      motion={false}
      closable={false}
      visible={visible}
      onCancel={() => setVisible(false)}>
      <Resizable
        defaultSize={{ width: width, height: '100%' }}
        handleClasses={{ left: 'left-handler' }}
        enable={{ left: true }}
        onResizeStop={(_e, _direction, _ref, d) => {
          initWidth.current += d.width;
        }}
        onResize={(_e, _direction, _ref, d) => {
          setWidth(d.width + initWidth.current);
        }}>
        <PanelEditContextProvider initPanel={(chart.model || {}) as PanelSetting}>
          <ChartDetail chart={chart} setVisible={setVisible} />
        </PanelEditContextProvider>
      </Resizable>
    </SideSheet>
  );
};

export default ChartDetailModal;
