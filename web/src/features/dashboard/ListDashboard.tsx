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
import {
  Button,
  Typography,
  Card,
  Col,
  Row,
  Table,
  Avatar,
  Dropdown,
  Rating,
  Collapse,
  List,
  Checkbox,
  RadioGroup,
  Radio,
  Input,
} from '@douyinfe/semi-ui';
import {
  IconSearchStroked,
  IconDeleteStroked,
  IconPlusStroked,
  IconStar,
  IconCopy,
  IconStarStroked,
  IconHandle,
} from '@douyinfe/semi-icons';
import React from 'react';
import { useQuery } from '@tanstack/react-query';
import { DashboardSrv } from '@src/services';
import { StatusTip } from '@src/components';
import { isEmpty } from 'lodash-es';
import { createSearchParams, useNavigate, useSearchParams } from 'react-router-dom';

const { Text } = Typography;

const ListDashboard: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const title = searchParams.get('title') || '';
  const ownership = searchParams.get('ownership') || '0';

  const { data, isLoading, isError, isFetching, error, refetch } = useQuery(
    ['fetch-dashboards', title, ownership],
    () => DashboardSrv.searchDashboards({ title: title, ownership: ownership })
  );

  return (
    <Card className="linsight-feature" bodyStyle={{ padding: 0 }}>
      <div style={{ margin: 16, display: 'flex', gap: 8 }}>
        <Input
          prefix={<IconSearchStroked />}
          placeholder="Filter dashboard"
          defaultValue={title}
          showClear
          onChange={(value: string) => {
            if (isEmpty(value)) {
              searchParams.delete('title');
            } else {
              searchParams.set('title', value);
            }
            setSearchParams(searchParams);
          }}
        />
        <Button icon={<IconPlusStroked />} onClick={() => navigate('/dashboard/edit')}>
          New
        </Button>
      </div>
      <Row>
        <Col span={4}>
          <Collapse defaultActiveKey={['1', '2', '3', '4']}>
            <Collapse.Panel itemKey="1" header="Favorite">
              <RadioGroup direction="vertical" defaultValue="2">
                <Radio value="1">Any</Radio>
                <Radio value="2">Yes</Radio>
                <Radio value="3">No</Radio>
              </RadioGroup>
            </Collapse.Panel>
            <Collapse.Panel itemKey="2" header="Ownership">
              <RadioGroup
                direction="vertical"
                defaultValue={ownership}
                onChange={(e: any) => {
                  searchParams.set('ownership', e.target.value);
                  setSearchParams(searchParams);
                }}>
                <Radio value="0">Any</Radio>
                <Radio value="1">Mine</Radio>
                <Radio value="2">Shared with me</Radio>
              </RadioGroup>
            </Collapse.Panel>
            <Collapse.Panel itemKey="3" header="Preset">
              <List split={false}>
                <List.Item main={<Checkbox>All Custom</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                <List.Item main={<Checkbox>All Integrations</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                <List.Item
                  main={<Checkbox>Frequently Viewed By You</Checkbox>}
                  style={{ padding: '4px 0px 4px 0px' }}
                />
              </List>
            </Collapse.Panel>
          </Collapse>
        </Col>
        <Col span={20}>
          <div style={{ marginLeft: 16, marginRight: 16, marginBottom: 16 }}>
            <Card bodyStyle={{ padding: 0, margin: 8 }} title={<Card.Meta title="All Dashboards" />}>
              <Table
                rowKey="uid"
                showHeader
                className="linsight"
                rowSelection={{ fixed: 'left', width: 48 }}
                size="small"
                dataSource={data?.dashboards || []}
                empty={
                  <StatusTip
                    isLoading={isLoading || isFetching}
                    isError={isError}
                    isEmpty={isEmpty(data?.dashboards)}
                    error={error}
                  />
                }
                pagination={
                  isEmpty(data?.dashboards)
                    ? false
                    : { total: data?.total || 0, pageSize: 20, style: { marginLeft: 8 } }
                }
                columns={[
                  {
                    title: (
                      <div style={{ display: 'flex', alignItems: 'center' }}>
                        <IconStar />
                      </div>
                    ),
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
                            onClick={() =>
                              navigate({
                                pathname: '/dashboard',
                                search: `${createSearchParams({
                                  d: r.uid,
                                })}`,
                              })
                            }>
                            {r.title}
                          </Text>
                        </>
                      );
                    },
                  },
                  {
                    title: 'Owner',
                    key: 'owner',
                    width: 120,
                    align: 'center',
                    dataIndex: 'owner',
                    render: () => {
                      return (
                        <Avatar color="blue" size="extra-small">
                          Df
                        </Avatar>
                      );
                    },
                  },
                  {
                    title: 'Modified At',
                    key: 'modified',
                    width: 120,
                    align: 'center',
                    dataIndex: 'modified',
                  },
                  {
                    title: 'Popularity',
                    key: 'popularity',
                    width: 120,
                    align: 'center',
                    dataIndex: 'owner',
                    render: () => {
                      return (
                        <Rating
                          className="p-rating"
                          size="small"
                          defaultValue={3}
                          character={<div style={{ width: 4, fontSize: 25 }}>I</div>}
                        />
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
                              <Dropdown.Item>Edit</Dropdown.Item>
                              <Dropdown.Item icon={<IconCopy />}>Clone</Dropdown.Item>
                              <Dropdown.Item
                                icon={<IconDeleteStroked />}
                                type="danger"
                                onClick={() => {
                                  DashboardSrv.deleteDashboard(r.uid);
                                  refetch();
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
  );
};

export default ListDashboard;
