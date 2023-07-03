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
import { Button, Form, Tag } from '@douyinfe/semi-ui';
import { useRequest } from '@src/hooks';
import { ComponentSrv } from '@src/services';
import { Component, OrgComponent } from '@src/types';
import React, { MutableRefObject, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { isEmpty, get, find } from 'lodash-es';
import { Icon, Notification } from '@src/components';
import { ApiKit } from '@src/utils';

const OrgComponent: React.FC<{ orgUid: string }> = (props) => {
  const { orgUid } = props;
  const [submitting, setSubmitting] = useState(false);
  const { result: navTree } = useRequest(['load_component_tree_org_cmp', orgUid], () => {
    return Promise.all([ComponentSrv.getComponentTree(), ComponentSrv.getOrgComponents(orgUid)]);
  });
  const [treeData, setTreeData] = useState<any[]>([]);
  const tree = useRef() as MutableRefObject<Map<string, any>>;
  const formApi = useRef<any>();
  const buildTree = useCallback((navTree: Component[], parent: Component | null): any[] => {
    return navTree.map((nav: Component) => {
      const item = {
        label: nav.label,
        key: nav.uid,
        icon: <Icon icon={nav.icon} style={{ marginRight: 8 }} />,
        value: nav.uid,
        parent: parent,
        role: nav.role,
        children: !isEmpty(nav.children) ? buildTree(nav.children, nav) : [],
      };
      tree.current.set(item.value, item);
      return item;
    });
  }, []);
  useMemo(() => {
    tree.current = new Map();
  }, []);

  useEffect(() => {
    tree.current.clear();
    const cmps = get(navTree, '[1]', []) || [];
    formApi.current.setValue(
      'components',
      cmps.map((c: OrgComponent) => c.componentUid)
    );
    setTreeData(buildTree(get(navTree, '[0]', []) || [], null));
  }, [navTree, buildTree]);

  const addOrgComponent = (uid: any, cmps: OrgComponent[]) => {
    const cmp = tree.current.get(uid);
    if (!cmp) {
      return;
    }
    if (cmp.parent) {
      addOrgComponent(cmp.parent.uid, cmps);
    }
    // not exist, add it
    if (!find(cmps, { componentUid: uid })) {
      cmps.push({ role: cmp.role, componentUid: uid });
    }
  };

  return (
    <Form
      style={{ marginTop: 24 }}
      onSubmit={async (values: any) => {
        const cmps: OrgComponent[] = [];
        (values.components || []).forEach((uid: string) => {
          addOrgComponent(uid, cmps);
        });
        try {
          setSubmitting(true);
          await ComponentSrv.saveOrgComponents(orgUid, cmps);
          Notification.success('Save components successfully!');
        } catch (err) {
          Notification.error(ApiKit.getErrorMsg(err));
        } finally {
          setSubmitting(false);
        }
      }}
      getFormApi={(api: any) => {
        formApi.current = api;
      }}>
      <Form.Section text="Component">
        <Form.TreeSelect
          field="components"
          noLabel
          expandAll
          filterTreeNode
          multiple
          leafOnly
          placeholder="Please select components"
          renderSelectedItem={(item, { index }) => ({
            content: (
              <Tag key={index} color="white" closable={false} style={{ padding: '0px 8px', height: 26 }}>
                {item.icon}
                {item.label}
              </Tag>
            ),

            isRenderInTag: false,
          })}
          treeData={treeData}
        />
      </Form.Section>
      <Button
        loading={submitting}
        onClick={() => {
          formApi.current.submitForm();
        }}>
        Save
      </Button>
    </Form>
  );
};

export default OrgComponent;
