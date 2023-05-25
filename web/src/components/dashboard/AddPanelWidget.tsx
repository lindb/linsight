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
import { Button, Card, Typography } from '@douyinfe/semi-ui';
import { IconClose } from '@douyinfe/semi-icons';
import { DashboardStore } from '@src/stores';
import React from 'react';
import { Icon } from '@src/components';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { DefaultVisualizationType } from '@src/constants';
import { PanelSetting } from '@src/types';

const { Text } = Typography;

const AddPanelWidget: React.FC<{
  panel: PanelSetting;
}> = (props) => {
  const { panel } = props;
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  return (
    <>
      <Card
        className="dashboard-panel add-panel-widget"
        bodyStyle={{ height: 'calc(100% - 46px)' }}
        header={
          <div className="panel-header">
            <div className="title">
              <Icon icon="icon-panel-add" />
              <Text>{panel.title}</Text>
            </div>
            <IconClose className="close" />
          </div>
        }>
        <div className="add-panel">
          <Button
            onClick={() => {
              panel.title = 'Panel title';
              panel.type = DefaultVisualizationType;
              searchParams.set('panel', panel.id);

              navigate({ pathname: '/dashboard/panel/edit', search: searchParams.toString() });
            }}>
            <div>
              <Icon icon="icon-line-bar" />
            </div>
            <div>Add a new panel</div>
          </Button>
          <Button
            onClick={() => {
              // nedd update add panel widget to row panel
              DashboardStore.updatePanelConfig(panel, {
                title: 'Row title',
                type: 'row',
                grid: { w: 24, h: 1, y: 0, x: 0 },
              });
            }}>
            <div>
              <Icon icon="icon-center" />
            </div>
            <div>Add a new row</div>
          </Button>
          <Button>
            <div>
              <Icon icon="icon-repo" />
            </div>
            <div>Add a panel from library</div>
          </Button>
        </div>
      </Card>
    </>
  );
};

export default AddPanelWidget;
