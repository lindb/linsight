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
import { Collapse, Form, Select, Typography } from '@douyinfe/semi-ui';
import { DatasourceStore } from '@src/stores';
import React, { useContext } from 'react';
import { FormatRepositoryInst, PanelSetting, VisualizationPlugin, VisualizationRepositoryInst } from '@src/types';
import { ObjectKit } from '@src/utils';
import { get } from 'lodash-es';
import { PanelEditContext } from '@src/contexts';

const { Text } = Typography;

/*
 * Panle basic information
 */
const PanelOptionsForm: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);

  return (
    <>
      <Form
        className="linsight-form linsight-panel-setting"
        layout="vertical"
        initValues={panel}
        onValueChange={(values: object) => {
          modifyPanel(values);
        }}>
        <Form.Input field="title" label="Title" />
        <Form.TextArea field="description" label="Description" rows={2} />
      </Form>
    </>
  );
};

/*
 * Visualization categories select form
 */
const VisualizationsForm: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const plugins = VisualizationRepositoryInst.getPlugins();

  const getDatasourceCategory = () => {
    const datasourceUID = get(panel, 'targets[0].datasource.uid');
    const ds = DatasourceStore.getDatasource(`${datasourceUID}`);
    return get(ds, 'plugin.category');
  };

  // get first target datasource category if exist, because all targets need have same datasource category.
  const datasourceCategory = getDatasourceCategory();

  return (
    <>
      <Select
        defaultValue={panel?.type}
        style={{ width: '100%' }}
        onChange={(v) => modifyPanel({ type: v } as object)}
        renderSelectedItem={(n: Record<string, any>) => {
          const plugin = VisualizationRepositoryInst.get(`${n.value}`);
          if (!plugin) {
            return null;
          }
          return (
            <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
              <img src={`${plugin.darkLogo}`} width={20} />
              <Text>{plugin.Name}</Text>
            </div>
          );
        }}>
        {plugins.map((plugin: VisualizationPlugin) => {
          if (datasourceCategory && plugin.category != datasourceCategory) {
            return;
          }
          return (
            <Select.Option key={plugin.Type} value={plugin.Type} showTick={false}>
              <img src={`${plugin.lightLogo}`} width={32} />
              <div style={{ marginLeft: 8 }}>
                <div>
                  <Text>{plugin.Name}</Text>
                </div>
                <Text size="small">{plugin.Description}</Text>
              </div>
            </Select.Option>
          );
        })}
      </Select>
    </>
  );
};

const StandardOptionsForm: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);
  return (
    <Form
      className="linsight-form linsight-panel-setting"
      layout="vertical"
      initValues={panel?.options}
      onValueChange={(values: object) => {
        modifyPanel({ options: values });
      }}>
      <Form.Cascader
        field="unit"
        label="Unit"
        placeholder="choose"
        style={{ width: '100%' }}
        filterTreeNode
        showClear
        leafOnly
        autoMergeValue={false}
        motion={false}
        treeData={FormatRepositoryInst.tree()}
        // treeData={FormatCategories}
        onChange={(value) => {
          console.log(value);
        }}
      />
      <Form.InputNumber field="decimals" label="Decimals" />
    </Form>
  );
};

const PanelSetting: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const plugin = VisualizationRepositoryInst.get(`${panel?.type}`);
  if (!plugin || !panel) {
    return null;
  }
  const OptionsEditor = plugin.components.OptionsEditor;
  return (
    <>
      <Collapse expandIconPosition="left" defaultActiveKey={['panelOptions', 'visualizations', 'standardOptions']}>
        <Collapse.Panel header="Panel options" itemKey="panelOptions">
          <PanelOptionsForm />
        </Collapse.Panel>
        <Collapse.Panel header="Visualizations" itemKey="visualizations">
          <VisualizationsForm />
        </Collapse.Panel>
        {OptionsEditor && (
          <OptionsEditor
            panel={ObjectKit.merge(plugin.getDefaultOptions(), panel)}
            onOptionsChange={(options: {}) => {
              console.error('option ccc', options);
              modifyPanel({ options: options });
            }}
          />
        )}
        <Collapse.Panel header="Standard options" itemKey="standardOptions">
          <StandardOptionsForm />
        </Collapse.Panel>
      </Collapse>
    </>
  );
};

export default PanelSetting;
