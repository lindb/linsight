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
import { IconChevronDown, IconRefresh, IconTick } from '@douyinfe/semi-icons';
import { Button, Dropdown, Input, Popover, Form, SplitButtonGroup, Typography, Space } from '@douyinfe/semi-ui';
import { isEmpty, find, filter } from 'lodash-es';
import React, { useState, useRef, MutableRefObject, useEffect } from 'react';
import moment from 'moment';
import { DateTimeFormat } from '@src/constants';
import { Icon } from '@src/components';
import { useSearchParams } from 'react-router-dom';
import { SearchParamKeys } from '@src/types';

const { Title } = Typography;
type QuickSelectItem = {
  title: string;
  value: string;
};
const defaultQuickItem = { title: 'Last 1 hour', value: 'now()-1h' };
const defaultAutoRefreshItem = { title: 'off', value: '' };
const quickSelectList: QuickSelectItem[] = [
  { title: 'Last 15 minutes', value: 'now()-15m' },
  { title: 'Last 30 minutes', value: 'now()-30m' },
  defaultQuickItem,
  { title: 'Last 3 hours', value: 'now()-3h' },
  { title: 'Last 6 hours', value: 'now()-6h' },
  { title: 'Last 12 hours', value: 'now()-12h' },
  { title: 'Last 1 day', value: 'now()-1d' },
  { title: 'Last 2 days', value: 'now()-2d' },
  { title: 'Last 3 days', value: 'now()-3d' },
  { title: 'Last 7 days', value: 'now()-7d' },
  { title: 'Last 15 days', value: 'now()-15d' },
  { title: 'Last 30 days', value: 'now()-30d' },
];
const autoRefreshList: QuickSelectItem[] = [
  {
    value: '',
    title: 'off',
  },
  { value: '10', title: `10s` },
  { value: '30', title: `30s` },
  { value: '60', title: `1m` },
  { value: '300', title: `5m` },
];

const TimePicker: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const from = searchParams.get(SearchParamKeys.From);
  const to = searchParams.get(SearchParamKeys.To);
  const refresh = searchParams.get(SearchParamKeys.Refresh);

  const formApi = useRef() as MutableRefObject<any>;
  const [quick, setQuick] = useState<QuickSelectItem | undefined>(defaultQuickItem);
  const [quickItems, setQuickItems] = useState<QuickSelectItem[]>(quickSelectList);
  const [visible, setVisible] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState<QuickSelectItem>(defaultAutoRefreshItem);
  const countDown = useRef<number>();
  const timeRangeVisible = useRef<boolean>(false);

  const buildCountDown = (interval: number) => {
    if (countDown.current) {
      clearInterval(countDown.current);
    }

    if (interval) {
      countDown.current = +setInterval(() => {
        // FIXME:
        //URLStore.forceChange();
      }, 1000 * interval);
    }
  };

  useEffect(() => {
    if (isEmpty(from)) {
      setQuick(defaultQuickItem);
    } else {
      const quickItem = find(quickSelectList, { value: `${from}` });
      setQuick(quickItem);
    }
  }, [from]);

  useEffect(() => {
    const refreshItem = find(autoRefreshList, { value: `${refresh}` });
    if (refreshItem && refreshItem.value !== '') {
      buildCountDown(parseInt(refreshItem.value));
    } else {
      clearInterval(countDown.current);
    }
    setAutoRefresh(refreshItem || defaultAutoRefreshItem);
  }, [refresh]);

  const renderQuickSelectItem = (items: QuickSelectItem[]) => {
    const SelectItems = items.map((item) => (
      <Dropdown.Item
        style={{ padding: 3 }}
        key={item.title}
        active={quick?.value == item.value}
        onClick={() => {
          setVisible(false);
          searchParams.delete(SearchParamKeys.To);
          searchParams.set(SearchParamKeys.From, `${item.value}`);
          setSearchParams(searchParams);
        }}>
        <IconTick
          style={{
            color: quick?.value !== item.value ? 'transparent' : 'inherit',
          }}
        />
        {item.title}
      </Dropdown.Item>
    ));
    return <Dropdown.Menu>{SelectItems}</Dropdown.Menu>;
  };

  /**
   * Render current selected time
   */
  function renderSelectedTime() {
    return (
      <Button icon={<Icon icon="iconclock" />} onClick={() => setVisible(true)}>
        {quick && quick.title}
        {!quick && `${from} ~ ${to ? `${to}` : 'now'}`}
      </Button>
    );
  }

  function renderTimeSelectPanel() {
    return (
      <Space style={{ width: 460, padding: 20 }} align="start">
        <div style={{ width: 230 }}>
          <Title heading={5}>Absolute time range</Title>
          <Form style={{ marginTop: 16 }} className="lin-form" getFormApi={(api: any) => (formApi.current = api)}>
            <Form.DatePicker
              field="from"
              type="dateTime"
              label="From"
              labelPosition="top"
              onOpenChange={(v) => (timeRangeVisible.current = v)}
              initValue={!quick ? from && new Date(`${from}`) : null}
            />
            <Form.DatePicker
              field="to"
              type="dateTime"
              labelPosition="top"
              label="To"
              onOpenChange={(v) => (timeRangeVisible.current = v)}
              initValue={to && new Date(`${to}`)}
            />
            <Button
              style={{ marginTop: 12 }}
              onClick={() => {
                setVisible(false);
                const from = formApi.current.getValue(SearchParamKeys.From);
                const to = formApi.current.getValue(SearchParamKeys.To);
                searchParams.set(SearchParamKeys.From, from ? moment(from.getTime()).format(DateTimeFormat) : '');
                searchParams.set(SearchParamKeys.To, to ? moment(to.getTime()).format(DateTimeFormat) : '');
              }}>
              Apply time range
            </Button>
          </Form>
        </div>
        <div
          style={{
            paddingLeft: 20,
            borderLeft: '1px solid var(--semi-color-border)',
          }}>
          <Title strong heading={6}>
            <Input
              placeholder="Search quick range"
              onChange={(val: string) => {
                const rs = filter(quickSelectList, (item: QuickSelectItem) => item.title.indexOf(val) >= 0);
                setQuickItems(rs);
              }}
            />
          </Title>
          {renderQuickSelectItem(quickItems)}
        </div>
      </Space>
    );
  }

  return (
    <>
      <Popover
        onClickOutSide={(_v) => {
          if (!timeRangeVisible.current) {
            // if click outside not date time range picker
            setVisible(false);
          }
        }}
        showArrow
        visible={visible}
        trigger="custom"
        position="bottom"
        content={renderTimeSelectPanel()}>
        {renderSelectedTime()}
      </Popover>
      <SplitButtonGroup>
        <Button
          icon={<IconRefresh />}
          onClick={() => {
            //FIXME:
          }}
        />
        <Dropdown
          trigger="click"
          showTick
          render={
            <Dropdown.Menu>
              {autoRefreshList.map((item) => (
                <Dropdown.Item
                  key={item.title}
                  active={item.value === autoRefresh.value}
                  onClick={() => {
                    searchParams.set(SearchParamKeys.Refresh, item.value);
                    setSearchParams(searchParams);
                  }}>
                  {item.title}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          }>
          {autoRefresh.value === '' ? (
            <Button icon={<IconChevronDown />} iconPosition="right" />
          ) : (
            <Button icon={<IconChevronDown />} iconPosition="right">
              {autoRefresh.title}
            </Button>
          )}
        </Dropdown>
      </SplitButtonGroup>
    </>
  );
};

export default TimePicker;
