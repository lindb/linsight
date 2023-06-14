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
import React, { useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Col, Empty, Form, Row, Select, Tag, Tree, Typography } from '@douyinfe/semi-ui';
import {
  IconSaveStroked,
  IconPlusStroked,
  IconPlusCircleStroked,
  IconDeleteStroked,
  IconTick,
} from '@douyinfe/semi-icons';
import { get, isEmpty, set, merge, has, remove, cloneDeep, sortBy, findIndex } from 'lodash-es';
import { Icon, Notification } from '@src/components';
import './menu.scss';
import * as IconFontsStyles from '@src/styles/icon-fonts/fonts.scss';
import { useRequest } from '@src/hooks';
import { NavSrv } from '@src/services';
import { v4 as uuidv4 } from 'uuid';
import { PlatformContext } from '@src/contexts';
import { ApiKit, ObjectKit } from '@src/utils';
import FormSlot from '@douyinfe/semi-ui/lib/es/form/slot';
import EmptyImg from '@src/images/empty.svg';

const { Text } = Typography;

const getSupportIconFonts = (): any[] => {
  const regex = /\.([a-zA-Z0-9_-]+)::before\s*\{/g;
  let match;
  const rs: any[] = [];

  while ((match = regex.exec(IconFontsStyles.default)) !== null) {
    const name = match[1];
    rs.push(name);
  }
  return sortBy(rs);
};

const MenuSetting: React.FC = () => {
  const { sync } = useContext(PlatformContext);
  const { result: navTree } = useRequest(['get_nav'], () => {
    return NavSrv.getNav();
  });
  const iconOptions = useRef<string[]>([]);
  const [treeData, setTreeData] = useState<any[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const formApi = useRef<any>();
  const tree = useRef<Map<string, any>>(new Map());
  const [currentParent, setCurrentParent] = useState<any>();
  const [editing, setEditing] = useState(false);

  useMemo(() => {
    iconOptions.current = getSupportIconFonts();
  }, []);

  const buildTree = useCallback((navTree: any, parent: any): any[] => {
    return navTree.map((nav: any) => {
      // if node has not '_key', set it
      if (!has(nav, '_key')) {
        const key = uuidv4();
        set(nav, '_key', key);
      }
      const key = get(nav, '_key');
      tree.current.set(key, nav);
      const item = {
        label: nav.label,
        key: key,
        _key: key,
        value: key,
        icon: <Icon icon={nav.icon} style={{ marginRight: 8 }} />,
        raw: nav,
        parent: parent,
        children: !isEmpty(nav.children) ? buildTree(nav.children, nav) : [],
      };
      return item;
    });
  }, []);

  const rebuildTree = () => {
    tree.current.clear();
    setTreeData(buildTree(navTree?.config || [], null));
  };

  useEffect(() => {
    tree.current.clear();
    setTreeData(buildTree(navTree?.config || [], null));
  }, [navTree, buildTree]);

  const onDrop = (info: any) => {
    const { dropToGap, node, dragNode } = info;
    const dropPos = node.pos.split('-');
    const dropPosition = info.dropPosition - Number(dropPos[dropPos.length - 1]);

    // 1. remove from source list
    const dragParent = dragNode.parent;
    const sourceList = dragParent ? dragParent.children : navTree.config;
    remove(sourceList, (o: any) => {
      // raw item raw using '_key'
      return get(o, '_key') === get(dragNode, '_key');
    });
    // 2. put into target list
    if (!dropToGap) {
      // if drop over other node, add this target node.
      node.raw.children = node.raw.children || [];
      node.raw.children.push(dragNode.raw);
    } else {
      const targetList = node.parent ? node.parent.children : navTree.config;
      const dropNodeIdx = findIndex(targetList, { _key: node._key });
      if (dropPosition === -1) {
        // insert to top
        targetList.splice(dropNodeIdx, 0, dragNode.raw);
      } else {
        // insert to bottom
        targetList.splice(dropNodeIdx + 1, 0, dragNode.raw);
      }
    }
    rebuildTree();
  };

  const addNewMenu = (target: any) => {
    setCurrentParent(target);
    formApi.current.setValues({}, { isOverride: true });
  };

  return (
    <Row className="menu-setting">
      <Col span={10} className="menu-tree">
        <div className="buttons">
          <Button
            icon={<IconPlusStroked />}
            type="tertiary"
            onClick={() => {
              setEditing(true);
              addNewMenu({ node: null, children: navTree.config });
            }}>
            Add
          </Button>
          <Button
            icon={<IconSaveStroked />}
            loading={submitting}
            onClick={async () => {
              try {
                setSubmitting(true);
                await NavSrv.updateNav(ObjectKit.removeUnderscoreProperties(navTree));
                sync();
                Notification.success('Menu save successfully!');
              } catch (err) {
                Notification.error(ApiKit.getErrorMsg(err));
              } finally {
                setSubmitting(false);
              }
            }}>
            Save
          </Button>
          <Button
            type="secondary"
            icon={<Icon icon="reset-setting" />}
            onClick={() => {
              // need clone default config, avoid modify default config
              navTree.config = cloneDeep(navTree.defaultConfig);
              rebuildTree();
            }}>
            Default
          </Button>
        </div>
        <Tree
          style={{ marginRight: 4 }}
          motion={false}
          draggable
          // expandedKeys={expand}
          onDrop={onDrop}
          treeData={treeData}
          renderLabel={(label: any, item: any) => (
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Typography.Text
                ellipsis={{ showTooltip: true }}
                style={{ flex: 1 }}
                onClick={() => {
                  setEditing(true);
                  setCurrentParent(null);
                  formApi.current.setValues(item.raw, { isOverride: true });
                }}>
                {label}
              </Typography.Text>
              <div>
                <Button
                  icon={<IconPlusCircleStroked />}
                  theme="borderless"
                  type="tertiary"
                  size="small"
                  onClick={() => {
                    setEditing(true);
                    item.raw.children = item.raw.children || [];
                    addNewMenu({ node: item.raw, children: item.raw.children });
                  }}
                />
                <Button
                  type="danger"
                  icon={<IconDeleteStroked />}
                  theme="borderless"
                  size="small"
                  onClick={() => {
                    const parent = item.parent;
                    // if no parent, is root node
                    const items = parent ? parent.children : navTree?.config;
                    remove(items, (o: any) => {
                      // raw item raw using '_key', tree item using 'key'
                      return get(o, '_key') === get(item, '_key');
                    });
                    rebuildTree();
                  }}
                />
              </div>
            </div>
          )}
        />
      </Col>
      <Col span={14} className="menu-form">
        {!editing && (
          <Empty
            title="Oops! No edit menu"
            style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginTop: 50 }}
            image={<img src={EmptyImg} style={{ width: 150, height: 150 }} />}
            darkModeImage={<img src={EmptyImg} style={{ width: 150, height: 150 }} />}
            layout="horizontal"
            description="Please select edit menu"
          />
        )}
        <Form
          style={{ display: editing ? 'block' : 'none', paddingLeft: 24 }}
          className="linsight-form"
          labelPosition="left"
          labelAlign="right"
          labelWidth={120}
          allowEmpty
          onSubmit={(values: object) => {
            const item = tree.current.get(get(values, '_key', ''));
            if (item) {
              merge(item, values);
            } else {
              set(values, '_key', uuidv4());
              currentParent.children.push(values);
              tree.current.set(get(values, '_key', ''), values);
              formApi.current.setValues(values, { isOverride: true });
            }
            rebuildTree();
          }}
          getFormApi={(api: any) => {
            formApi.current = api;
          }}>
          {currentParent && (
            <FormSlot label="Parent">
              <Tag size="large">{currentParent.node ? currentParent.node.label : 'Root'}</Tag>
            </FormSlot>
          )}
          <Form.Input field="label" label="Label" rules={[{ required: true, message: 'Label is required' }]} />
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
            {(iconOptions.current || []).map((option: any) => {
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
          <Form.Slot>
            <Button
              icon={<IconTick />}
              onClick={() => {
                formApi.current.submitForm();
              }}>
              Apply
            </Button>
          </Form.Slot>
        </Form>
      </Col>
    </Row>
  );
};

export default MenuSetting;
