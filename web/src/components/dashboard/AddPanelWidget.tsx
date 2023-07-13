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
import { Button, Card, Checkbox, Modal, Transfer, Typography } from '@douyinfe/semi-ui';
import { IconClose, IconHandle } from '@douyinfe/semi-icons';
import { DashboardStore } from '@src/stores';
import React, { MutableRefObject, useRef, useState } from 'react';
import { Icon, IntegrationIcon, StatusTip, VisualizationIcon } from '@src/components';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Chart, PanelSetting } from '@src/types';
import { useRequest } from '@src/hooks';
import { ChartSrv } from '@src/services';
import './panel.scss';
import { isEmpty } from 'lodash-es';

const { Text } = Typography;

const AddPanelFromChartRepo: React.FC<{
  visible: boolean;
  setVisible: (v: boolean) => void;
}> = (props) => {
  const { visible, setVisible } = props;
  const { result, loading, error } = useRequest(['search-charts'], () => ChartSrv.searchCharts({}));
  const selectedCharts = useRef() as MutableRefObject<Chart[]>;

  return (
    <>
      <Modal
        title="Add panel from chart repository"
        closeOnEsc
        className="add-panel-widget"
        closable={false}
        motion={false}
        maskClosable={false}
        size="large"
        visible={visible}
        onOk={() => {
          DashboardStore.addCharts(selectedCharts.current);
          setVisible(false);
        }}
        okText="Add to dashboard"
        onCancel={() => {
          setVisible(false);
        }}>
        <Transfer
          draggable
          dataSource={(result?.charts || []).map((c: Chart) => {
            return { value: c.uid, key: c.uid, ...c };
          })}
          emptyContent={{ left: <StatusTip isLoading={loading} isEmpty={isEmpty(result?.charts)} error={error} /> }}
          onChange={(_values: any, items: any[]) => {
            selectedCharts.current = items;
          }}
          renderSelectedItem={(item: any) => {
            const { sortableHandle } = item;
            const DragHandle = sortableHandle(() => <IconHandle className={`semi-right-item-drag-handler`} />);

            return (
              <div className="selected-item chart-selected-item" key={item.uid}>
                <DragHandle />
                <div className="chart-item">
                  <IntegrationIcon integration={item.integration} style={{ fontSize: 28 }} />
                  <div className="info">
                    <Text>{item.title}</Text>
                    <Text type="tertiary" size="small">
                      {item.description || 'N/A'}
                    </Text>
                  </div>
                </div>
                <VisualizationIcon type={item.type} />
                <IconClose onClick={item.onRemove} />
              </div>
            );
          }}
          renderSourceItem={(item: any) => {
            return (
              <div className="chart-list" key={item.uid}>
                <Checkbox
                  onChange={() => {
                    item.onChange();
                  }}
                  value={item.uid}
                  checked={item.checked}
                  className="chart-item">
                  <IntegrationIcon integration={item.integration} style={{ fontSize: 28 }} />
                  <div className="info">
                    <Text>{item.title}</Text>
                    <Text type="tertiary" size="small">
                      {item.description || 'N/A'}
                    </Text>
                  </div>
                  <VisualizationIcon type={item.type} />
                </Checkbox>
              </div>
            );
          }}
        />
      </Modal>
    </>
  );
};

/*
 * Add panel widget in dashboard.
 */
const AddPanelWidget: React.FC<{
  panel: PanelSetting;
}> = (props) => {
  const { panel } = props;
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [visible, setVisible] = useState(false);
  return (
    <>
      <Card
        className="dashboard-panel add-panel-widget"
        bodyStyle={{ height: 'calc(100% - 46px)' }}
        header={
          <div className="panel-header">
            <div className="title">
              <Icon icon="panel-add" />
              <Text>{panel.title}</Text>
            </div>
            <IconClose className="close" onClick={() => DashboardStore.deletePanel(panel)} />
          </div>
        }>
        <div className="add-panel">
          <Button
            onClick={() => {
              searchParams.set('panel', `${panel.id}`);
              navigate({ pathname: '/dashboard/panel/edit', search: searchParams.toString() });
            }}>
            <div>
              <Icon icon="line-bar" />
            </div>
            <div>Add a new panel</div>
          </Button>
          <Button
            onClick={() => {
              // nedd update add panel widget to row panel
              DashboardStore.updatePanelConfig(panel, {
                title: 'Row title',
                type: 'row',
                gridPos: { w: 24, h: 1, y: 0, x: 0 },
              });
            }}>
            <div>
              <Icon icon="center" />
            </div>
            <div>Add a new row</div>
          </Button>
          <Button
            onClick={() => {
              setVisible(true);
            }}>
            <div>
              <Icon icon="repo" />
            </div>
            <div>Add a panel from chart repo.</div>
          </Button>
        </div>
      </Card>
      {visible && <AddPanelFromChartRepo visible={visible} setVisible={setVisible} />}
    </>
  );
};

export default AddPanelWidget;
