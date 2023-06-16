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
import { PlatformContext } from '@src/contexts';
import { MenuStore } from '@src/stores';
import React, { useContext, useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { isEmpty, get, upperFirst } from 'lodash-es';
import { Button, Dropdown, Modal, Nav, Table, Tag, Typography } from '@douyinfe/semi-ui';
import { IconChevronRightStroked, IconUser, IconMoon, IconSun } from '@douyinfe/semi-icons';
import { Notification } from '@src/components';
import Icon from '../common/Icon';
import Sider from '@douyinfe/semi-ui/lib/es/layout/Sider';
import Logo from '@src/images/logo.svg';
import { ThemeType, UserOrg } from '@src/types';
import { UserSrv } from '@src/services';
import { matchPath } from 'react-router';
import { v4 as uuidv4 } from 'uuid';
import { useRequest } from '@src/hooks';
import { ApiKit } from '@src/utils';

const { Text } = Typography;

const SwitchOrg: React.FC<{
  userUid: string;
  currentOrgUid: string;
  visible: boolean;
  setVisible: (v: boolean) => void;
}> = (props) => {
  const { userUid, currentOrgUid, visible, setVisible } = props;
  const { sync } = useContext(PlatformContext);
  const { result: orgList, loading } = useRequest(['get_org_list_for_current_user', userUid], () => {
    return UserSrv.getOrgListForUser(userUid);
  });
  const [submitting, setSubmitting] = useState(false);

  return (
    <Modal
      motion={false}
      footer={null}
      title="Switch organization"
      visible={visible}
      bodyStyle={{ paddingBottom: 24 }}
      closeOnEsc
      onCancel={() => setVisible(false)}>
      <Table
        bordered
        size="small"
        rowKey="orgUid"
        loading={loading}
        dataSource={orgList || []}
        pagination={false}
        columns={[
          {
            title: 'Org. name',
            dataIndex: 'orgName',
          },
          {
            title: 'Role',
            dataIndex: 'role',
          },
          {
            render: (_text: string, r: UserOrg, _index: number) => {
              if (r.orgUid === currentOrgUid) {
                return <Tag color="orange">Current</Tag>;
              }
              return (
                <Button
                  loading={submitting}
                  icon={<Icon icon="switch" />}
                  onClick={async () => {
                    setSubmitting(true);
                    try {
                      await UserSrv.switchOrg(r.orgUid);
                      Notification.success('Switch organization successfully!');
                      sync();
                      setVisible(false);
                    } catch (err) {
                      Notification.error(ApiKit.getErrorMsg(err));
                    } finally {
                      setSubmitting(false);
                    }
                  }}>
                  Switch
                </Button>
              );
            },
          },
        ]}
      />
    </Modal>
  );
};

const FeatureMenu: React.FC = () => {
  const { boot, collapsed, toggleCollapse, toggleTheme, theme } = useContext(PlatformContext);
  const navigate = useNavigate();
  const location = useLocation();
  const [menus, setMenus] = useState([]);
  const currentOrg = get(boot, 'user.org');
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    MenuStore.setCurrentMenu(location.pathname);
  }, [location]);

  const selectMenus = (): string[] => {
    let menu: string[] = [];
    const path = location.pathname;
    (boot.navTree || []).forEach((item: any) => {
      if (!isEmpty(item.path)) {
        if (matchPath(`${item.path}/*`, path)) {
          menu.push(item.path || item.label);
          return;
        }
      }
      (item.children || []).forEach((subItem: any) => {
        if (matchPath(`${subItem.path}/*`, path)) {
          menu.push(subItem.path || subItem.label);
          return;
        }
      });
    });
    return menu;
  };

  useEffect(() => {
    const renderMenus = (menus: any) => {
      return (menus || []).map((item: any) => {
        if (item.children) {
          return (
            <Nav.Sub
              key={uuidv4()}
              level={0}
              itemKey={item.path || item.label}
              icon={<Icon icon={item.icon} style={{ fontSize: 20 }} />}
              text={item.label}>
              {item.children.map((child: any) => (
                <Nav.Item
                  level={1}
                  key={uuidv4()}
                  icon={<Icon icon={child.icon} style={{ fontSize: 20 }} />}
                  text={child.label}
                  itemKey={child.path || child.label}
                  onClick={() => navigate(child.path)}
                />
              ))}
            </Nav.Sub>
          );
        }

        if (collapsed) {
          return (
            <Nav.Sub
              level={0}
              key={uuidv4()}
              itemKey={item.path || item.label}
              icon={<Icon icon={item.icon} style={{ fontSize: 20 }} />}
              text={item.label}>
              <Nav.Item
                level={1}
                icon={<Icon icon={item.icon} style={{ fontSize: 20 }} />}
                itemKey={item.path || item.label}
                key={uuidv4()}
                text={item.label}
                onClick={() => navigate(item.path)}
              />
            </Nav.Sub>
          );
        }
        return (
          <Nav.Item
            level={0}
            icon={<Icon icon={item.icon} style={{ fontSize: 20 }} />}
            itemKey={item.path || item.label}
            key={uuidv4()}
            text={item.label}
            onClick={() => navigate(item.path)}
          />
        );
      });
    };
    setMenus(renderMenus(boot.navTree));
  }, [boot.navTree, collapsed, navigate]);

  return (
    <>
      {visible && (
        <SwitchOrg
          userUid={get(boot, 'user.uid')}
          currentOrgUid={currentOrg.uid}
          visible={visible}
          setVisible={(v: boolean) => {
            setVisible(v);
          }}
        />
      )}
      <div className="nav-menu-no-icon"></div>
      <Button
        style={{
          left: collapsed ? 48 : 208,
        }}
        className="sider-collapse-btn"
        type="tertiary"
        size="small"
        onClick={toggleCollapse}
        icon={<IconChevronRightStroked rotate={collapsed ? 0 : 180} />}
      />

      <Sider className="linsight-sider">
        <Nav
          subNavMotion={false}
          limitIndent={true}
          isCollapsed={collapsed}
          getPopupContainer={(): any => {
            return document.querySelector('.nav-menu-no-icon');
          }}
          selectedKeys={selectMenus()}
          style={{ maxWidth: 220, height: '100%' }}
          header={{
            logo: <img src={Logo} onClick={() => navigate('/')} />,
            text: 'insight',
          }}
          footer={{
            children: (
              <div className="linsight-nav-footer">
                <Dropdown
                  spacing={8}
                  position="right"
                  render={
                    <Dropdown.Menu className="linsight-user">
                      <Dropdown.Item disabled>{get(boot, 'user.name')}</Dropdown.Item>
                      <Dropdown.Divider />
                      {currentOrg && (
                        <Dropdown.Item
                          onClick={() => {
                            setVisible(true);
                          }}>
                          <span style={{ marginRight: 8 }}>Organization:</span>
                          <Tag color="orange">{currentOrg.name}</Tag>
                        </Dropdown.Item>
                      )}
                      <Dropdown.Item onClick={() => navigate('/user/profile')}>Your profile</Dropdown.Item>
                      <Dropdown.Item onClick={() => navigate('/user/password')}>Change password</Dropdown.Item>
                      <Dropdown.Item icon={theme !== ThemeType.Dark ? <IconMoon /> : <IconSun />} onClick={toggleTheme}>
                        {upperFirst(theme)}
                      </Dropdown.Item>
                      <Dropdown.Divider />
                      <Dropdown.Item
                        type="danger"
                        icon={<Icon icon="signout" />}
                        onClick={async () => {
                          await UserSrv.logout();
                          navigate('/login');
                        }}>
                        Sign out
                      </Dropdown.Item>
                    </Dropdown.Menu>
                  }>
                  <div
                    className="user-info"
                    style={collapsed ? { justifyContent: 'center' } : { justifyContent: 'start', paddingLeft: 10 }}>
                    <IconUser size="large" />
                    {!collapsed && (
                      <Text ellipsis style={{ width: 160 }}>
                        {get(boot, 'user.name')}
                      </Text>
                    )}
                  </div>
                </Dropdown>
              </div>
            ),
          }}>
          {menus}
        </Nav>
      </Sider>
    </>
  );
};

export default FeatureMenu;
