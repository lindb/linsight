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
import { IconDeleteStroked } from '@douyinfe/semi-icons';
import { Button, Modal, Typography } from '@douyinfe/semi-ui';
import { DatasourceSrv } from '@src/services';
import { DatasourceStore } from '@src/stores';
import { Notification } from '@src/components';
import { ApiKit } from '@src/utils';
import React, { useState } from 'react';
const { Text } = Typography;

const DeleteDatasourceButton: React.FC<{ uid: string; name: string; text?: string; onCompleted: () => void }> = (
  props
) => {
  const { uid, name, text, onCompleted } = props;
  const [visible, setVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  return (
    <>
      <Button
        type="danger"
        icon={<IconDeleteStroked />}
        loading={submitting}
        onClick={() => {
          setVisible(true);
        }}>
        {text}
      </Button>
      <Modal
        title="Delete datasource"
        motion={false}
        visible={visible}
        onCancel={() => setVisible(false)}
        footer={
          <>
            <Button
              type="tertiary"
              onClick={() => {
                setVisible(false);
              }}>
              Cancel
            </Button>
            <Button
              type="danger"
              theme="solid"
              onClick={async () => {
                if (uid) {
                  setSubmitting(true);
                  try {
                    await DatasourceSrv.deleteDatasource(uid);
                    await DatasourceStore.syncDatasources();
                    onCompleted();
                    setVisible(false);
                  } catch (err) {
                    Notification.error(ApiKit.getErrorMsg(err));
                  } finally {
                    setSubmitting(false);
                  }
                }
              }}>
              Delete
            </Button>
          </>
        }>
        Are you sure you want to delete [<Text type="danger">{name}</Text>] datasource?
      </Modal>
    </>
  );
};
export default DeleteDatasourceButton;
