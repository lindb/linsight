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
import { Modal } from '@douyinfe/semi-ui';
import React from 'react';

const UnlinkChart: React.FC<{ visible: boolean; setVisible: (v: boolean) => void; unlinkChart: () => void }> = (
  props
) => {
  const { visible, setVisible, unlinkChart } = props;
  return (
    <Modal
      title="Do you really want to unlink this panel?"
      visible={visible}
      closeOnEsc
      okButtonProps={{ type: 'danger' }}
      okText="Yes,Unlink"
      onOk={() => unlinkChart()}
      onCancel={() => setVisible(false)}>
      If you unlink this panel, you can edit it without affecting other dashboards. But once you make changes, you
      cannot go back to the original reusable panel.
    </Modal>
  );
};

export default UnlinkChart;
