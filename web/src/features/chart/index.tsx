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
import React, { useRef, useState } from 'react';
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
  Dropdown,
  Modal,
  Tag,
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconPlusStroked, IconSearchStroked, IconHandle, IconDeleteStroked } from '@douyinfe/semi-icons';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { StatusTip, Notification, AddToDashboard, IntegrationIcon, VisualizationIcon } from '@src/components';
import { isEmpty } from 'lodash-es';
import { useRequest } from '@src/hooks';
import { Chart, VisualizationRepositoryInst } from '@src/types';
import './chart.scss';
import ChartDetailModal from './ChartDetail';
import { ApiKit } from '@src/utils';
import ListDashboardByChart from './ListDashboardByChart';
const { Text } = Typography;

const ListChart: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const title = searchParams.get('title') || '';
  const ownership = searchParams.get('ownership') || '0';
  const [visible, setVisible] = useState(false);
  const [dashboardVisible, setDashboardVisible] = useState(false);
  const [deleteVisible, setDeleteVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [chartSelected, setChartSelected] = useState(false);
  const currentChart = useRef<Chart>();
  const currentSelectedRows = useRef<any>();

  const { result, loading, error, refetch } = useRequest(['fetch-charts', title, ownership], () =>
    ChartSrv.searchCharts({ title: title, ownership: ownership })
  );

  return (
    <>
      <ChartDetailModal visible={visible} setVisible={setVisible} chart={currentChart.current || {}} />
      <ListDashboardByChart
        visible={dashboardVisible}
        setVisible={setDashboardVisible}
        chartUid={currentChart.current?.uid || ''}
      />
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
              <Card
                bodyStyle={{ padding: 0, margin: 8 }}
                title={
                  <Card.Meta
                    title={
                      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                        <div>All charts</div>
                        <AddToDashboard
                          disabled={!chartSelected}
                          getCharts={() => {
                            return currentSelectedRows.current;
                          }}
                        />
                      </div>
                    }
                  />
                }>
                <Table
                  rowKey="uid"
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
                  rowSelection={{
                    onChange: (selectedRowKeys: any, selectedRows: any) => {
                      currentSelectedRows.current = selectedRows;
                      setChartSelected(!isEmpty(selectedRowKeys));
                    },
                  }}
                  columns={[
                    {
                      title: 'Title',
                      key: 'title',
                      align: 'left',
                      dataIndex: 'title',
                      render: (_text: any, r: any, _index: any) => {
                        return (
                          <div
                            className="dashboard-title"
                            onClick={() => {
                              currentChart.current = r;
                              setVisible(true);
                            }}>
                            <IntegrationIcon integration={r.integration} style={{ fontSize: 20 }} />
                            <Text link>{r.title}</Text>
                          </div>
                        );
                      },
                    },
                    {
                      title: 'Description',
                      key: 'description',
                      align: 'left',
                      dataIndex: 'description',
                    },
                    {
                      title: 'Visualization',
                      key: 'type',
                      dataIndex: 'type',
                      align: 'center',
                      render: (_text: any, r: any, _index: any) => {
                        return <VisualizationIcon type={r.type} />;
                      },
                    },
                    {
                      title: 'Dashboards',
                      key: 'dashboards',
                      align: 'left',
                      dataIndex: 'dashboards',
                      render: (_text: any, r: any, _index: any) => {
                        return (
                          <Tag
                            style={{ cursor: 'pointer' }}
                            size="large"
                            onClick={() => {
                              currentChart.current = r;
                              setDashboardVisible(true);
                            }}>
                            {r.dashboards}
                          </Tag>
                        );
                      },
                    },
                    {
                      key: 'action',
                      width: 50,
                      align: 'center',
                      dataIndex: 'action',
                      render: (_text: any, r: any, _index: any) => {
                        return (
                          <Dropdown
                            render={
                              <Dropdown.Menu>
                                <Dropdown.Item
                                  icon={<IconDeleteStroked />}
                                  type="danger"
                                  onClick={() => {
                                    currentChart.current = r;
                                    setDeleteVisible(true);
                                  }}>
                                  Delete
                                </Dropdown.Item>
                              </Dropdown.Menu>
                            }>
                            <IconHandle style={{ cursor: 'pointer' }} />
                          </Dropdown>
                        );
                      },
                    },
                  ]}
                />
              </Card>
            </div>
          </Col>
        </Row>
      </Card>
      <Modal
        title={
          <div>
            Delete [<Text type="danger">{currentChart.current?.title}</Text>] chart
          </div>
        }
        motion={false}
        visible={deleteVisible}
        onCancel={() => setDeleteVisible(false)}
        footer={
          <>
            <Button
              type="tertiary"
              onClick={() => {
                setDeleteVisible(false);
              }}>
              Cancel
            </Button>
            <Button
              type="danger"
              theme="solid"
              loading={submitting}
              onClick={async () => {
                setSubmitting(true);
                try {
                  if (currentChart.current && currentChart.current.uid) {
                    await ChartSrv.deleteChart(currentChart.current.uid);
                    setDeleteVisible(false);
                    refetch();
                    Notification.success('Chart deleted!');
                  }
                } catch (err) {
                  console.warn('delete chart error', err);
                  Notification.error(ApiKit.getErrorMsg(err));
                } finally {
                  setSubmitting(false);
                }
              }}>
              Yes
            </Button>
          </>
        }>
        Are you sure you want to remove this chart?
      </Modal>
    </>
  );
};

export default ListChart;
