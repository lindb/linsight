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
import { useRequest } from '@src/hooks';
import { ComponentSrv } from '@src/services';
import { Component, RoleList } from '@src/types';
import { isEmpty } from 'lodash-es';
import { Button, Col, Divider, Empty, Form, Row, Tree, Typography } from '@douyinfe/semi-ui';
import { Icon, Notification } from '@src/components';
import EmptyImg from '@src/images/empty.svg';
import './component.scss';
import { IconRefresh, IconSaveStroked } from '@douyinfe/semi-icons';
import { ApiKit } from '@src/utils';

const OrgComponent: React.FC = () => {
  const { result: navTree, refetch } = useRequest(['load_org_component_tree'], () => {
    return ComponentSrv.getOrgComponentTree();
  });
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);
  const [treeData, setTreeData] = useState<any[]>([]);
  const [editing, setEditing] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [selectItem, setSelectItem] = useState<any>({});
  const formApi = useRef<any>();

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

  useEffect(() => {
    setTreeData(buildTree(navTree || [], null));
  }, [navTree, buildTree]);

  return (
    <Row className="menu-setting">
      <Col span={10} className="menu-tree">
        <div className="buttons">
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
          treeData={treeData}
          renderLabel={(label: any, item: any) => (
            <div style={{ display: 'flex', alignItems: 'center' }}>
              <Typography.Text
                ellipsis={{ showTooltip: true }}
                style={{ flex: 1 }}
                onClick={() => {
                  setSelectItem(item.raw);
                  formApi.current.setValues(item.raw, { isOverride: true });
                  setEditing(true);
                }}>
                {label}
              </Typography.Text>
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
              await ComponentSrv.updateRoleOfOrgComponent([{ componentUid: selectItem.uid, role: values.role }]);
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
          <Form.Slot label="Label">
            <Icon icon={selectItem.icon} />
            <Typography.Text style={{ marginLeft: 4 }}>{selectItem.label}</Typography.Text>
          </Form.Slot>
          <Form.Select
            label="Role"
            field="role"
            optionList={RoleList}
            rules={[{ required: true, message: 'Label is required' }]}
          />
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

export default OrgComponent;
