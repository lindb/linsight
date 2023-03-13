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
import { Form, Card, Row, Col, Button, Typography, Divider, Layout } from '@douyinfe/semi-ui';
import { IconGithubLogo, IconFacebook } from '@douyinfe/semi-icons';
import React, { useRef, useState } from 'react';
import LoginImg from '@src/images/login.svg';
import { Footer, Icon, Notification } from '@src/components';
import { UserSrv } from './services';
import { useNavigate } from 'react-router-dom';
import { ApiKit } from './utils';
const { Text } = Typography;

const Login: React.FC = () => {
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const formApi = useRef<any>();
  const login = async () => {
    setSubmitting(true);
    try {
      if (formApi.current) {
        const api = formApi.current;
        const pass = await api.validate();
        if (pass) {
          await UserSrv.login(api.getValues());
          navigate('/');
        }
      }
    } catch (err) {
      Notification.error(ApiKit.getErrorMsg(err));
    } finally {
      setSubmitting(false);
    }
  };
  return (
    <Layout>
      <Layout.Content>
        <div className="login">
          <Row gutter={48}>
            <Col offset={6} span={6}>
              <img src={`${LoginImg}`} style={{ height: 'auto', maxWidth: '100%' }} />
            </Col>
            <Col span={6}>
              <Card>
                <Form
                  className="login-form"
                  onKeyDown={(e: any) => {
                    if (e.keyCode === 13) {
                      login();
                    }
                  }}
                  disabled={submitting}
                  getFormApi={(api: any) => (formApi.current = api)}>
                  <Form.Section text="Sign In to Linsight">
                    <Form.Input
                      field="username"
                      label="Email or username"
                      rules={[{ required: true, message: 'Email or username is required' }]}
                      placeholder="Email or usename"
                    />
                    <Form.Input
                      mode="password"
                      field="password"
                      label="Password"
                      placeholder="Password"
                      rules={[{ required: true, message: 'Password is required' }]}
                    />
                    <Form.Slot>
                      <Button
                        style={{ width: '100%' }}
                        theme="solid"
                        onClick={() => login()}
                        disabled={submitting}
                        loading={submitting}>
                        {submitting ? 'Logging in' : 'Log in'}
                      </Button>
                    </Form.Slot>
                    <Form.Slot>
                      <Text link>Forgot password?</Text>
                    </Form.Slot>
                  </Form.Section>
                </Form>
                <Divider align="center">Or</Divider>
                <div style={{ gap: 8, marginTop: 8, display: 'flex', alignItems: 'center' }}>
                  <Icon icon="icon-google" style={{ fontSize: 24 }} />
                  <IconGithubLogo size="extra-large" />
                  <IconFacebook size="extra-large" />
                </div>
              </Card>
            </Col>
          </Row>
        </div>
      </Layout.Content>
      <Footer />
    </Layout>
  );
};

export default Login;
