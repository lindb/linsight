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
import { ChartSrv } from '@src/services';
import React, { useContext, useRef, useState } from 'react';
import {
  Card,
  TagInput,
  Row,
  Col,
  Collapse,
  RadioGroup,
  Radio,
  List,
  Checkbox,
  Button,
  Table,
  Typography,
  SideSheet,
} from '@douyinfe/semi-ui';
import {
  IconPlusStroked,
  IconClose,
  IconSearchStroked,
  IconStar,
  IconStarStroked,
  IconSaveStroked,
} from '@douyinfe/semi-icons';
import { createSearchParams, useNavigate, useSearchParams } from 'react-router-dom';
import { DatasourceSelectForm, Notification, Icon, MetricExplore, StatusTip } from '@src/components';
import { isEmpty, get } from 'lodash-es';
import { useRequest } from '@src/hooks';
import { Chart, DatasourceInstance, PanelSetting } from '@src/types';
import { PanelEditContext, PanelEditContextProvider } from '@src/contexts';
import { DatasourceStore } from '@src/stores';
import './chart.scss';
import { ApiKit } from '@src/utils';
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

const ListChart: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const title = searchParams.get('title') || '';
  const ownership = searchParams.get('ownership') || '0';
  const [visible, setVisible] = useState(false);
  const currentChart = useRef<PanelSetting>();

  const { result, loading, error } = useRequest(['fetch-charts', title, ownership], () =>
    ChartSrv.searchCharts({ title: title, ownership: ownership })
  );

  return (
    <>
      <ChartDetailModal visible={visible} setVisible={setVisible} chart={currentChart.current || {}} />
      <Card className="linsight-feature" bodyStyle={{ padding: 0 }}>
        <div style={{ margin: 16, display: 'flex', gap: 8 }}>
          <TagInput prefix={<IconSearchStroked />} placeholder="Filter charts" defaultValue={['team:monitor']} />
          <Button icon={<IconPlusStroked />} onClick={() => navigate('/explore')}>
            New
          </Button>
        </div>
        <Row>
          <Col span={4}>
            <Collapse defaultActiveKey={['1', '2', '3', '4']}>
              <Collapse.Panel itemKey="1" header="Favorite">
                <RadioGroup direction="vertical" defaultValue="1">
                  <Radio value="1">Any</Radio>
                  <Radio value="2">Yes</Radio>
                  <Radio value="3">No</Radio>
                </RadioGroup>
              </Collapse.Panel>
              <Collapse.Panel itemKey="3" header="Metric List">
                <List split={false}>
                  <List.Item main={<Checkbox>Gauge</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Count</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Max</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>Histogram</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                </List>
              </Collapse.Panel>
              <Collapse.Panel itemKey="3" header="Preset">
                <List split={false}>
                  <List.Item main={<Checkbox>All Custom</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                  <List.Item main={<Checkbox>All Builtin</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                </List>
              </Collapse.Panel>
            </Collapse>
          </Col>
          <Col span={20}>
            <div style={{ marginLeft: 16, marginRight: 16, marginBottom: 16 }}>
              <Card bodyStyle={{ padding: 0, margin: 8 }} title={<Card.Meta title="All Charts" />}>
                <Table
                  rowKey="id"
                  showHeader
                  className="linsight"
                  size="small"
                  dataSource={result?.charts || []}
                  empty={<StatusTip isLoading={loading} isEmpty={isEmpty(result?.charts)} error={error} />}
                  pagination={
                    isEmpty(result?.charts)
                      ? false
                      : { total: result?.total || 0, pageSize: 20, style: { marginLeft: 8 } }
                  }
                  columns={[
                    {
                      title: <IconStar />,
                      key: 'favorite',
                      width: 40,
                      align: 'center',
                      dataIndex: 'status',
                      render: (_text: any, r: any, _index: any) => {
                        if (r.favorite) {
                          return <IconStar style={{ cursor: 'pointer' }} />;
                        }
                        return <IconStarStroked style={{ cursor: 'pointer' }} />;
                      },
                    },
                    {
                      title: 'Title',
                      key: 'title',
                      align: 'left',
                      dataIndex: 'title',
                      render: (_text: any, r: any, _index: any) => {
                        return (
                          <>
                            {r.type && <i style={{ marginRight: 4 }} className={`devicon-${r.type}-plain colored`} />}
                            <Text
                              link
                              onClick={() => {
                                currentChart.current = r;
                                setVisible(true);
                              }}>
                              {r.title}
                            </Text>
                          </>
                        );
                      },
                    },
                    {
                      title: 'Description',
                      key: 'desc',
                      align: 'left',
                      dataIndex: 'desc',
                    },
                  ]}
                />
              </Card>
            </div>
          </Col>
        </Row>
      </Card>
    </>
  );
};

export default ListChart;
