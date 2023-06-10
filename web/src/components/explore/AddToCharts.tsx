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
import React, { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { Button, Form, Modal } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { PanelSetting } from '@src/types';
import { ApiKit, ObjectKit } from '@src/utils';
import { ChartSrv } from '@src/services';
import { Icon, Notification } from '@src/components';

/*
 * Add metric explore to chart repository.
 */
const AddToCharts = forwardRef((_props: {}, ref) => {
  const [visible, setVisible] = useState(false);
  const chartOptions = useRef<PanelSetting>();
  const formApi = useRef<any>();
  const [submitting, setSubmitting] = useState(false);

  useImperativeHandle(ref, () => ({
    setOptions(options: PanelSetting) {
      chartOptions.current = options;
    },
  }));

  const saveChart = async (values: any) => {
    setSubmitting(true);
    try {
      await ChartSrv.createChart(ObjectKit.merge(chartOptions.current || {}, values));
      Notification.success('Save chart successfully');
      setVisible(false);
    } catch (err) {
      Notification.error(ApiKit.getErrorMsg(err));
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <>
      <Button type="tertiary" onClick={() => setVisible(true)} icon={<Icon icon="repo" />}>
        Add to chart
      </Button>
      <Modal
        size="medium"
        title="Add to chart repository"
        visible={visible}
        motion={false}
        onCancel={() => setVisible(false)}
        footer={
          <>
            <Button type="tertiary" onClick={() => setVisible(false)}>
              Cancel
            </Button>
            <Button
              icon={<IconSaveStroked />}
              loading={submitting}
              type="primary"
              theme="solid"
              onClick={() => {
                if (!formApi.current) {
                  return;
                }
                formApi.current.submitForm();
              }}>
              Save
            </Button>
          </>
        }>
        <Form
          className="linsight-form"
          allowEmpty
          getFormApi={(api: any) => (formApi.current = api)}
          onSubmit={(values: any) => {
            saveChart(values);
          }}>
          <Form.Input field="title" label="Title" rules={[{ required: true, message: 'Title is required' }]} />
          <Form.TextArea field="description" label="Description" />
        </Form>
      </Modal>
    </>
  );
});

AddToCharts.displayName = 'AddToCharts';

export default AddToCharts;
