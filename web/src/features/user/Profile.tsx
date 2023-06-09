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
import { Button, Form, Radio } from '@douyinfe/semi-ui';
import { IconSaveStroked } from '@douyinfe/semi-icons';
import { PlatformContext } from '@src/contexts';
import { UserSrv } from '@src/services';
import React, { useContext, useEffect, useRef, useState } from 'react';
import { Notification } from '@src/components';
import { useRequest } from '@src/hooks';
import { Preference } from '@src/types';
import { ApiKit } from '@src/utils';

const Profile: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const { sync } = useContext(PlatformContext);
  const { result: preference } = useRequest(['get_preference'], () => {
    return UserSrv.getPreference();
  });
  const preferenceFormApi = useRef<any>();

  useEffect(() => {
    preferenceFormApi.current.setValues(preference);
  }, [preference]);

  return (
    <div>
      <Form
        className="linsight-form"
        getFormApi={(api: any) => {
          preferenceFormApi.current = api;
        }}
        onSubmit={async (values: Preference) => {
          try {
            setSubmitting(true);
            await UserSrv.savePreference(values);
            sync();
            Notification.success('Preference save successfully!');
          } catch (err) {
            Notification.error(ApiKit.getErrorMsg(err));
          } finally {
            setSubmitting(false);
          }
        }}>
        <Form.Section text="Preferences">
          <Form.Input field="homePage" label="Home Page" />
          <Form.RadioGroup field="theme" label="Theme" type="button">
            <Radio value="default">Default</Radio>
            <Radio value="light">Light</Radio>
            <Radio value="dark">Dark</Radio>
            <Radio value="system">System</Radio>
          </Form.RadioGroup>
          <Form.Switch field="collapsed" label="Collapse sider" />
          <Form.Slot>
            <Button type="primary" icon={<IconSaveStroked />} htmlType="submit" loading={submitting}>
              Save
            </Button>
          </Form.Slot>
        </Form.Section>
      </Form>
    </div>
  );
};

export default Profile;
