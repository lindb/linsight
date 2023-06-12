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
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Col, Form, Row, Select, Tree, Typography } from '@douyinfe/semi-ui';
import { IconSaveStroked, IconPlusStroked, IconDeleteStroked } from '@douyinfe/semi-icons';
import { get, isEmpty, set, merge } from 'lodash-es';
import { Icon } from '@src/components';
import './menu.scss';
import IconFontsStyles from '@src/styles/icon-fonts/fonts.scss';
import { useRequest } from '@src/hooks';
import { NavSrv } from '@src/services';
const { Text } = Typography;

const getSupportIconFonts = (): any[] => {
  const regex = /\.([a-zA-Z0-9_-]+)::before\s*\{/g;
  let match;
  const rs: any[] = [];

  while ((match = regex.exec(IconFontsStyles)) !== null) {
    const name = match[1];
    rs.push(name);
  }
  return rs;
};

const MenuSetting: React.FC = () => {
  const { result: navTree } = useRequest(['get_nav'], () => {
    return NavSrv.getNav();
  });
  const iconOptions = getSupportIconFonts();
  const [treeData, setTreeData] = useState<any[]>([]);
  const formApi = useRef<any>();
  const tree = useRef<Map<string, any>>(new Map());

  const buildTree = useCallback((navTree: any, parent: any): any[] => {
    return navTree.map((nav: any) => {
      const key = nav.path || nav.text;
      tree.current.set(key, nav);
      set(nav, '_key', key);
      const item = {
        label: nav.text,
        key: key,
        value: key,
        icon: <Icon icon={nav.icon} style={{ marginRight: 8 }} />,
        values: nav,
        parent: parent,
        children: !isEmpty(nav.children) ? buildTree(nav.children, nav) : [],
      };
      return item;
    });
  }, []);

  useEffect(() => {
    const tree = buildTree(navTree?.config || [], null);
    console.error(tree);
    setTreeData(tree);
  }, [navTree, buildTree]);

  return (
    <Row className="menu-setting">
      <Col span={6} className="menu-tree">
        <div className="buttons">
          <Button icon={<IconPlusStroked />} type="tertiary">
            Add
          </Button>
          <Button icon={<IconSaveStroked />} onClick={() => NavSrv.updateNav(navTree)}>
            Save
          </Button>
        </div>
        <Tree
          expandAll
          treeData={treeData}
          renderLabel={(label: any, item: any) => (
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Typography.Text
                ellipsis={{ showTooltip: true }}
                style={{ width: 'calc(100% - 32px)' }}
                onClick={() => {
                  formApi.current.setValues(item.values, { isOverride: true });
                }}>
                {label}
              </Typography.Text>
              {!item.parent && (
                <Button
                  icon={<IconPlusStroked />}
                  theme="borderless"
                  type="tertiary"
                  size="small"
                  onClick={() => {
                    if (!item.values.children) {
                      item.values.children = [];
                    }
                    item.values.children.push({ text: 'empty', value: 'value..' });
                    setTreeData(buildTree(navTree, null));
                  }}
                />
              )}
            </div>
          )}
        />
      </Col>
      <Col span={18} className="menu-form">
        <Form
          labelPosition="left"
          allowEmpty
          onValueChange={(values: object) => {
            const item = tree.current.get(get(values, '_key', ''));
            if (item) {
              merge(item, values);
            }
            setTreeData(buildTree(navTree.config, null));
          }}
          getFormApi={(api: any) => {
            formApi.current = api;
          }}>
          <Form.Input field="text" label="Text" />
          <Form.Input field="path" label="Path" />
          <Form.Select
            filter
            field="icon"
            label="Icon"
            renderSelectedItem={(n: Record<string, any>) => {
              return (
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <Icon icon={n.value} />
                  <Text>{n.value}</Text>
                </div>
              );
            }}>
            {iconOptions.map((option: any) => {
              return (
                <Select.Option key={option} value={option} showTick={false}>
                  <Icon icon={option} />
                  <div style={{ marginLeft: 8 }}>
                    <Text>{option}</Text>
                  </div>
                </Select.Option>
              );
            })}
          </Form.Select>
          <Button type="danger" icon={<IconDeleteStroked />}>
            Delete
          </Button>
        </Form>
      </Col>
    </Row>
  );
};

export default MenuSetting;
