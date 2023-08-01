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
  Card,
  Checkbox,
  Col,
  Collapse,
  Dropdown,
  List,
  Radio,
  RadioGroup,
  Row,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconPlusStroked,
  IconStarStroked,
  IconStar,
  IconCopy,
  IconHandle,
  IconDeleteStroked,
} from '@douyinfe/semi-icons';
import { usePagination, useRequest } from '@src/hooks';
import { DashboardSrv, TagSrv } from '@src/services';
import React, { useEffect, useState } from 'react';
import { createSearchParams, useNavigate, useSearchParams } from 'react-router-dom';
import { includes, isEmpty, set, union } from 'lodash-es';
import { StatusTip, Notification, IntegrationIcon, SearchFilterInput, AttributeProps } from '@src/components';
import { ColumnProps } from '@douyinfe/semi-ui/lib/es/table';
import { Dashboard, SearchDashboard } from '@src/types';
import { ApiKit, ColorKit, StringKit } from '@src/utils';
const { Text } = Typography;

const attributes: AttributeProps[] = [
  {
    type: 'input',
    value: 'title',
    label: 'Title',
  },
  {
    type: 'select',
    value: 'tags',
    label: 'Tag',
    multiple: true,
    remote: (prefix: string) => {
      return TagSrv.findTags(prefix);
    },
  },
  {
    type: 'select',
    value: 'favorite',
    label: 'Favorite',
    options: [
      { value: '1', label: 'Any' },
      { value: '2', label: 'Yes' },
      { value: '3', label: 'No' },
    ],
  },
  {
    type: 'select',
    value: 'ownership',
    label: 'Ownership',
    options: [
      { value: '0', label: 'Any' },
      { value: '1', label: 'Mine' },
    ],
  },
];

const getSearchParams = (searchParams: URLSearchParams): SearchDashboard => {
  const value: SearchDashboard = {};
  attributes.forEach((attr: AttributeProps) => {
    if (!searchParams.has(attr.value)) {
      return;
    }
    if (attr.multiple) {
      set(value, attr.value, union(searchParams.getAll(attr.value)));
    } else {
      set(value, attr.value, searchParams.get(attr.value));
    }
  });
  return value;
};

const DashboardSearch: React.FC<{ searchOnly?: boolean }> = (props) => {
  const { searchOnly } = props;
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const { currentPage, pageSize, offset, onChange } = usePagination(1, 20);
  const [params, setParams] = useState<SearchDashboard>(() => {
    return getSearchParams(searchParams);
  });

  const {
    result: data,
    loading,
    error,
    refetch,
  } = useRequest(['fetch-dashboards', params, offset, pageSize], () => {
    return DashboardSrv.searchDashboards({ limit: pageSize, offset: offset, ...params });
  });

  useEffect(() => {
    setParams(getSearchParams(searchParams));
  }, [searchParams]);

  const getColumns = () => {
    const columns: ColumnProps[] = [
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
        render: (_text: any, r: Dashboard, _index: any) => {
          if (r.isStarred) {
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
        render: (_text: any, r: Dashboard, _index: any) => {
          return (
            <div
              className="dashboard-title"
              onClick={() => {
                navigate({
                  pathname: '/dashboard',
                  search: `${createSearchParams({
                    d: `${r.uid}`,
                  })}`,
                });
              }}>
              <IntegrationIcon integration={r.integration} style={{ fontSize: 16 }} />
              <Text link>{r.title}</Text>
            </div>
          );
        },
      },
      {
        title: 'Tags',
        key: 'tags',
        align: 'left',
        width: '40%',
        dataIndex: 'tags',
        render: (_text: any, r: Dashboard, _index: any) => {
          if (isEmpty(r.tags)) {
            return null;
          }
          return (
            <div style={{ display: 'flex', gap: 2 }}>
              {(r.tags || []).map((tag: string) => (
                <Tooltip key={tag} content="Add tag filter">
                  <Tag
                    style={{ backgroundColor: ColorKit.getColor(StringKit.hashcode(tag)), cursor: 'pointer' }}
                    type="solid"
                    onClick={() => {
                      const tags = searchParams.getAll('tags');
                      if (!includes(tags, tag)) {
                        searchParams.append('tags', tag);
                        setSearchParams(searchParams);
                      }
                    }}>
                    {tag}
                  </Tag>
                </Tooltip>
              ))}
            </div>
          );
        },
      },
    ];
    if (!searchOnly) {
      columns.push({
        key: 'action',
        width: 50,
        align: 'center',
        dataIndex: 'action',
        render: (_text: any, r: Dashboard, _index: any) => {
          return (
            <Dropdown
              render={
                <Dropdown.Menu>
                  <Dropdown.Item>Edit</Dropdown.Item>
                  <Dropdown.Item icon={<IconCopy />}>Clone</Dropdown.Item>
                  <Dropdown.Item
                    icon={<IconDeleteStroked />}
                    type="danger"
                    onClick={async () => {
                      try {
                        await DashboardSrv.deleteDashboard(`${r.uid}`);
                        refetch();
                        Notification.success('Dashboard deleted!');
                      } catch (err) {
                        console.warn('delete dashboard error', err);
                        Notification.error(ApiKit.getErrorMsg(err));
                      }
                    }}>
                    Delete
                  </Dropdown.Item>
                </Dropdown.Menu>
              }>
              <IconHandle style={{ cursor: 'pointer' }} />
            </Dropdown>
          );
        },
      });
    }
    return columns;
  };

  return (
    <Card className="linsight-feature" bodyStyle={{ padding: 0 }}>
      <div style={{ margin: 16, display: 'flex', gap: 8 }}>
        <SearchFilterInput placeholder="Filter dashboard" values={params} attributes={attributes} />
        {!searchOnly && (
          <Button size="large" icon={<IconPlusStroked />} onClick={() => navigate('/dashboard/new')}>
            New
          </Button>
        )}
      </div>
      <Row>
        <Col span={4}>
          <Collapse defaultActiveKey={['1', '2', '3', '4']}>
            <Collapse.Panel itemKey="1" header="Favorite">
              <RadioGroup direction="vertical" value={params.favorite || '1'}>
                <Radio value="1">Any</Radio>
                <Radio value="2">Yes</Radio>
                <Radio value="3">No</Radio>
              </RadioGroup>
            </Collapse.Panel>
            <Collapse.Panel itemKey="2" header="Ownership">
              <RadioGroup
                direction="vertical"
                value={params.ownership || '0'}
                onChange={(e: any) => {
                  searchParams.set('ownership', e.target.value);
                  setSearchParams(searchParams);
                }}>
                <Radio value="0">Any</Radio>
                <Radio value="1">Mine</Radio>
              </RadioGroup>
            </Collapse.Panel>
            <Collapse.Panel itemKey="3" header="Preset">
              <List split={false}>
                <List.Item main={<Checkbox>All Custom</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
                <List.Item main={<Checkbox>All Integrations</Checkbox>} style={{ padding: '4px 0px 4px 0px' }} />
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
                empty={<StatusTip isLoading={loading} isEmpty={isEmpty(data?.dashboards)} error={error} />}
                pagination={
                  isEmpty(data?.dashboards)
                    ? false
                    : {
                        total: data?.total || 0,
                        pageSize: pageSize,
                        currentPage: currentPage,
                        style: { marginLeft: 8 },
                        showSizeChanger: true,
                        onChange: (currentPage: number, pageSize: number) => {
                          onChange(currentPage, pageSize);
                        },
                      }
                }
                columns={getColumns()}
                scroll={{ y: 400 }}
              />
            </Card>
          </div>
        </Col>
      </Row>
    </Card>
  );
};

export default DashboardSearch;
