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
import { isEmpty, find, filter, get } from 'lodash-es';
import React, { useState, useRef, MutableRefObject, useEffect, useContext, useCallback } from 'react';
import moment from 'moment';
import {
  AutoRefreshList,
  DateTimeFormat,
  DefaultAutoRefreshItem,
  DefaultQuickItem,
  QuickSelectList,
} from '@src/constants';
import { Icon } from '@src/components';
import { useSearchParams } from 'react-router-dom';
import { QuickSelectItem, SearchParamKeys } from '@src/types';
import { VariableContext } from '@src/contexts';

const { Title } = Typography;

const TimePicker: React.FC = () => {
  const { from, to, refresh, refreshInterval } = useContext(VariableContext);
  const [searchParams, setSearchParams] = useSearchParams();

  const formApi = useRef() as MutableRefObject<any>;
  const [quick, setQuick] = useState<QuickSelectItem | undefined>(DefaultQuickItem);
  const [quickItems, setQuickItems] = useState<QuickSelectItem[]>(QuickSelectList);
  const [visible, setVisible] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState<QuickSelectItem>(DefaultAutoRefreshItem);
  const countDown = useRef<number>();
  const timeRangeVisible = useRef<boolean>(false);

  const buildCountDown = useCallback(
    (interval: number) => {
      if (countDown.current) {
        clearInterval(countDown.current);
      }

      if (interval) {
        countDown.current = +setInterval(() => {
          refresh();
        }, 1000 * interval);
      }
    },
    [refresh]
  );

  useEffect(() => {
    if (isEmpty(from)) {
      setQuick(DefaultQuickItem);
    } else {
      const quickItem = find(QuickSelectList, { value: `${from}` });
      setQuick(quickItem);
    }
  }, [from]);

  useEffect(() => {
    const refreshItem = find(AutoRefreshList, { title: `${refreshInterval}` });
    if (refreshItem && refreshItem.value !== '') {
      buildCountDown(parseInt(refreshItem.value));
    } else {
      clearInterval(countDown.current);
    }
    setAutoRefresh(refreshItem || DefaultAutoRefreshItem);
  }, [refreshInterval, buildCountDown]);

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
          <Form
            style={{ marginTop: 16 }}
            className="lin-form"
            getFormApi={(api: any) => (formApi.current = api)}
            onSubmit={(values: any) => {
              const from = get(values, SearchParamKeys.From);
              const to = get(values, SearchParamKeys.To);
              searchParams.set(SearchParamKeys.From, from ? moment(from.getTime()).format(DateTimeFormat) : '');
              searchParams.set(SearchParamKeys.To, to ? moment(to.getTime()).format(DateTimeFormat) : '');
              setSearchParams(searchParams);
            }}>
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
                formApi.current.submitForm();
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
                const rs = filter(QuickSelectList, (item: QuickSelectItem) => item.title.indexOf(val) >= 0);
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
            refresh();
          }}
        />
        <Dropdown
          trigger="click"
          showTick
          render={
            <Dropdown.Menu>
              {AutoRefreshList.map((item: QuickSelectItem) => (
                <Dropdown.Item
                  key={item.title}
                  active={item.value === autoRefresh.value}
                  onClick={() => {
                    if (item.value === '') {
                      searchParams.delete(SearchParamKeys.Refresh);
                    } else {
                      searchParams.set(SearchParamKeys.Refresh, item.title);
                    }
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
