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
import React, { useContext, useState } from 'react';
import { Button, Card, Form, Radio } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { UserSrv } from '@src/services';
import { PlatformContext } from '@src/contexts';

const EditUser: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const { boot } = useContext(PlatformContext);
  const { user } = boot;
  return (
    <Card className="linsight-feature">
      <Form
        initValues={user}
        labelPosition="left"
        labelAlign="right"
        labelWidth={150}
        onSubmit={async (values: any) => {
          try {
            setSubmitting(true);
            await UserSrv.saveUser(values);
          } finally {
            // FIXME: handle err
            setSubmitting(false);
          }
        }}>
        <Form.Input field="email" label="Email" disabled />
        <Form.Input field="name" label="Name" />
        <Form.RadioGroup field="preference.theme" label="Theme" type="button">
          <Radio value="default">Default</Radio>
          <Radio value="light">Light</Radio>
          <Radio value="dark">Dark</Radio>
        </Form.RadioGroup>
        <Form.Slot>
          <Button type="primary" icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
            Save
          </Button>
        </Form.Slot>
      </Form>
    </Card>
  );
};

export default EditUser;
