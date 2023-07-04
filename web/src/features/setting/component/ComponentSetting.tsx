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
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Col, Divider, Empty, Form, Row, Select, Tag, Tree, Typography } from '@douyinfe/semi-ui';
import {
  IconSaveStroked,
  IconPlusStroked,
  IconPlusCircleStroked,
  IconDeleteStroked,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { get, isEmpty, remove, sortBy, findIndex } from 'lodash-es';
import { Icon, Notification } from '@src/components';
import './component.scss';
import * as IconFontsStyles from '@src/styles/icon-fonts/fonts.scss?inline';
import { useRequest } from '@src/hooks';
import { ComponentSrv } from '@src/services';
import { ApiKit } from '@src/utils';
import EmptyImg from '@src/images/empty.svg';
import { Component, Feature, FeatureRepositoryInst, RoleList } from '@src/types';

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

const ComponentSetting: React.FC = () => {
  const { result: navTree, refetch } = useRequest(['load_component_tree'], () => {
    return ComponentSrv.getComponentTree();
  });
  const iconOptions = useRef<string[]>([]);
  const [treeData, setTreeData] = useState<any[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const formApi = useRef<any>();
  const tree = useRef<Map<string, any>>(new Map());
  const [currentParent, setCurrentParent] = useState<any>();
  const [editing, setEditing] = useState(false);
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);

  useMemo(() => {
    iconOptions.current = getSupportIconFonts();
  }, []);

  const buildTree = useCallback((navTree: Component[], parent: Component | null): any[] => {
    return navTree.map((nav: Component) => {
      const item = {
        label: nav.label,
        uid: nav.uid,
        key: nav.uid,
        value: nav.uid,
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
    setTreeData(buildTree(navTree || [], null));
  };

  useEffect(() => {
    tree.current.clear();
    setTreeData(buildTree(navTree || [], null));
  }, [navTree, buildTree]);

  const onDrop = (info: any) => {
    const { dropToGap, node, dragNode } = info;
    const dropPos = node.pos.split('-');
    const dropPosition = info.dropPosition - Number(dropPos[dropPos.length - 1]);

    // 1. remove from source list
    const dragParent = dragNode.parent;
    const sourceList = dragParent ? dragParent.children : navTree;
    console.error(sourceList, dragNode);
    remove(sourceList, (o: any) => {
      // raw item raw using '_key'
      return get(o, 'uid') === get(dragNode, 'uid');
    });
    var targetList = [];
    // 2. put into target list
    if (!dropToGap) {
      // if drop over other node, add this target node.
      dragNode.raw.parentUID = node.raw.uid;
      node.raw.children = node.raw.children || [];
      targetList = node.row.children;
      node.raw.children.push(dragNode.raw);
    } else {
      dragNode.raw.parentUID = node.parent ? node.parent.uid : '';
      targetList = node.parent ? node.parent.children : navTree;
      const dropNodeIdx = findIndex(targetList, { uid: node.uid });
      if (dropPosition === -1) {
        // insert to top
        targetList.splice(dropNodeIdx, 0, dragNode.raw);
      } else {
        // insert to bottom
        targetList.splice(dropNodeIdx + 1, 0, dragNode.raw);
      }
    }
    rebuildTree();
    ComponentSrv.sortComponents(targetList)
      .then(() => {
        refetch();
        Notification.success('Component save successfully!');
      })
      .catch((err) => {
        Notification.error(ApiKit.getErrorMsg(err));
      });
  };

  const addNewMenu = (target: any) => {
    setCurrentParent(target);
    formApi.current.setValues({ parentUid: target ? target.uid : '' }, { isOverride: true });
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
              addNewMenu(null);
            }}>
            Add
          </Button>
          <Button
            icon={<IconRefresh />}
            type="tertiary"
            onClick={() => {
              refetch();
            }}
          />
        </div>
        <Divider />
        <Tree
          style={{ marginRight: 4 }}
          motion={false}
          draggable
          onExpand={(keys: string[]) => setExpandedKeys(keys)}
          expandedKeys={expandedKeys}
          onDrop={onDrop}
          treeData={treeData}
          renderLabel={(label: any, item: any) => (
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Typography.Text
                ellipsis={{ showTooltip: true }}
                style={{ flex: 1 }}
                onClick={() => {
                  setEditing(true);
                  setCurrentParent(item.parent);
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
                    addNewMenu(item.raw);
                  }}
                />
                <Button
                  type="danger"
                  icon={<IconDeleteStroked />}
                  theme="borderless"
                  size="small"
                  onClick={async () => {
                    try {
                      await ComponentSrv.deleteComponentByUID(item.uid);
                      if (item.uid === formApi.current.getValue('uid')) {
                        // if delete current edit cmp, need clear edit flag
                        setEditing(false);
                      }
                      refetch();
                      Notification.success('Component deleted!');
                    } catch (err) {
                      Notification.error(ApiKit.getErrorMsg(err));
                    }
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
            title="Oops! No edit component"
            style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', marginTop: 50 }}
            image={<img src={EmptyImg} style={{ width: 150, height: 150 }} />}
            darkModeImage={<img src={EmptyImg} style={{ width: 150, height: 150 }} />}
            layout="horizontal"
            description="Please select edit component"
          />
        )}
        <Form
          style={{ display: editing ? 'block' : 'none', paddingLeft: 24 }}
          className="linsight-form"
          labelPosition="left"
          labelAlign="right"
          labelWidth={120}
          allowEmpty
          onSubmit={async (values: Component) => {
            try {
              setSubmitting(true);
              if (values.uid) {
                await ComponentSrv.updateComponent(values);
              } else {
                const uid = await ComponentSrv.createComponent(values);
                values.uid = uid;
                formApi.current.setValues(values, { isOverride: true });
              }
              refetch();
              Notification.success('Component save successfully!');
            } catch (err) {
              Notification.error(ApiKit.getErrorMsg(err));
            } finally {
              setSubmitting(false);
            }
          }}
          getFormApi={(api: any) => {
            formApi.current = api;
          }}>
          <Form.Slot label="Parent">
            <Tag size="large">{currentParent ? currentParent.label : 'Root'}</Tag>
          </Form.Slot>
          <Form.Input field="label" label="Label" rules={[{ required: true, message: 'Label is required' }]} />
          <Form.Select
            label="Role"
            field="role"
            optionList={RoleList}
            rules={[{ required: true, message: 'Label is required' }]}
          />
          <Form.Input field="path" label="Path" />
          <Form.Select
            label="Component"
            field="component"
            renderSelectedItem={(n: Record<string, any>) => {
              const feature = FeatureRepositoryInst.getFeature(`${n.value}`);
              if (!feature) {
                return null;
              }
              return (
                <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                  <Text>{feature.label}</Text>
                </div>
              );
            }}>
            {FeatureRepositoryInst.getFeatures().map((feature: Feature) => {
              return (
                <Select.Option key={feature.key} value={feature.key} showTick={false}>
                  <div style={{ marginLeft: 8 }}>
                    <div>
                      <Text strong>{feature.label}</Text>
                    </div>
                    <Text size="small">{feature.desc}</Text>
                  </div>
                </Select.Option>
              );
            })}
          </Form.Select>
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
              icon={<IconSaveStroked />}
              loading={submitting}
              onClick={() => {
                formApi.current.submitForm();
              }}>
              Save
            </Button>
          </Form.Slot>
        </Form>
      </Col>
    </Row>
  );
};

export default ComponentSetting;
