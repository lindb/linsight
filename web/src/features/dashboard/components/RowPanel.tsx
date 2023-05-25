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
import React, { useEffect, useRef, useState } from 'react';
import { PanelSetting } from '@src/types';
import { Button, Form, Modal, Typography } from '@douyinfe/semi-ui';
import { IconSettingStroked, IconDeleteStroked, IconChevronDownStroked, IconHandle } from '@douyinfe/semi-icons';
import { DashboardStore } from '@src/stores';
import { get } from 'lodash-es';
import classNames from 'classnames';

const { Text } = Typography;

/*
 * Row panel in dashboard
 */
const RowPanel: React.FC<{ panel: PanelSetting }> = (props) => {
  const { panel } = props;
  const [visible, setVisible] = useState(false);
  const [deleteVisible, setDeleteVisible] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    return get(panel, 'collapsed', false);
  });
  const [showActions, setShowActions] = useState(false);
  const formApi = useRef<any>();
  useEffect(() => {
    DashboardStore.collapseRow(panel, collapsed);
  }, [collapsed, panel]);
  const cls = classNames('dashboard-row-panel', { collapsed: collapsed });
  return (
    <>
      <div
        className={cls}
        onMouseEnter={() => {
          setShowActions(true);
        }}
        onMouseLeave={() => {
          setShowActions(false);
        }}>
        <div className="title" onClick={() => setCollapsed(!collapsed)}>
          <IconChevronDownStroked size="small" rotate={collapsed ? -90 : 0} />
          <Text className="text" strong>
            {panel.title}
          </Text>
        </div>
        <div className="actions" style={{ visibility: showActions ? 'visible' : 'hidden' }}>
          <Button theme="borderless" type="tertiary" icon={<IconSettingStroked />} onClick={() => setVisible(true)} />
          <Button
            theme="borderless"
            type="tertiary"
            icon={<IconDeleteStroked />}
            onClick={() => setDeleteVisible(true)}
          />
        </div>
        {collapsed && (
          <>
            <div className="drag-handle" />
            <IconHandle className="drag grid-drag-handle" size="large" />
          </>
        )}
      </div>
      <Modal
        title="Row options"
        visible={visible}
        motion={false}
        onCancel={() => setVisible(false)}
        onOk={() => {
          if (formApi.current) {
            formApi.current.submitForm();
          }
        }}>
        <Form
          initValues={panel}
          getFormApi={(api: any) => {
            formApi.current = api;
          }}
          onSubmit={(values: any) => {
            DashboardStore.updatePanelConfig(panel, values);
            setVisible(false);
          }}>
          <Form.Input field="title" label="Title" rules={[{ required: true, message: 'Title is required' }]} />
        </Form>
      </Modal>
      <Modal
        title="Delete row"
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
              onClick={() => {
                DashboardStore.deleteRowAndChildren(panel);
                setDeleteVisible(false);
              }}>
              Yes
            </Button>
            <Button
              onClick={() => {
                DashboardStore.deletePanel(panel);
                setDeleteVisible(false);
              }}>
              Delete row only
            </Button>
          </>
        }>
        Are you sure you want to remove this row and all its panels?
      </Modal>
    </>
  );
};

export default RowPanel;
