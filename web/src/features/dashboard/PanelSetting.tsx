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
import React, { useContext, useEffect, useRef, useState } from 'react';
import {
  ArrayField,
  Avatar,
  Button,
  Collapse,
  Form,
  Modal,
  Radio,
  Select,
  SplitButtonGroup,
  Typography,
} from '@douyinfe/semi-ui';
import { IconPlusStroked, IconDeleteStroked, IconLink, IconUnlink } from '@douyinfe/semi-icons';
import { DatasourceStore } from '@src/stores';
import {
  FieldConfig,
  FormatRepositoryInst,
  PanelSetting,
  Thresholds,
  Threshold,
  VisualizationPlugin,
  VisualizationRepositoryInst,
} from '@src/types';
import { ColorKit, ObjectKit } from '@src/utils';
import { get, set, isEmpty, size, isNil, orderBy, maxBy, has, cloneDeep, unset } from 'lodash-es';
import { PanelEditContext } from '@src/contexts';
import { IntegrationIcon } from '@src/components';

const { Text } = Typography;

const ChartRepoOptionsForm: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const [visible, setVisible] = useState(false);

  const unlinkChart = () => {
    unset(panel, 'libraryPanel');
    modifyPanel(panel, true);
  };

  if (panel.libraryPanel) {
    return (
      <>
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
        <SplitButtonGroup>
          <Button icon={<IconUnlink />} type="tertiary" onClick={() => setVisible(true)} />
          <Button
            icon={<IntegrationIcon integration={panel.integration} style={{ fontSize: 18 }} />}
            type="tertiary"
            onClick={() => setVisible(true)}>
            <Text>{panel.libraryPanel?.name}</Text>
          </Button>
        </SplitButtonGroup>
      </>
    );
  }

  return (
    <Button icon={<IconLink />} type="tertiary">
      Link
    </Button>
  );
};

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
        allowEmpty
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
        onChange={(v: any) => {
          const newCfg = ObjectKit.merge(VisualizationRepositoryInst.getDefaultOptions(v), panel);
          newCfg.type = v;
          modifyPanel(newCfg);
        }}
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
  const fieldConfig: FieldConfig = cloneDeep(get(panel, 'fieldConfig.defaults', {}));
  const formApi = useRef<any>();
  useEffect(() => {
    if (formApi.current) {
      const unit = get(fieldConfig, 'unit', '');
      const formatter = FormatRepositoryInst.get(unit);
      set(fieldConfig, '_unit', [formatter.Category.category, formatter.Category.value]);
      formApi.current.setValues(fieldConfig, { isOverride: true });
    }
  }, [formApi, fieldConfig]);
  return (
    <Form
      className="linsight-form linsight-panel-setting"
      layout="vertical"
      getFormApi={(api: any) => (formApi.current = api)}
      onValueChange={(values: object) => {
        set(values, 'unit', get(values, '_unit[1]', ''));
        modifyPanel({
          fieldConfig: {
            defaults: set(values, 'unit', get(values, '_unit[1]', '')),
          },
        });
      }}>
      <Form.Cascader
        field="_unit"
        label="Unit"
        placeholder="choose"
        filterTreeNode
        showClear
        leafOnly
        autoMergeValue={false}
        motion={false}
        treeData={FormatRepositoryInst.tree()}
      />
      <Form.InputNumber field="min" label="Min" />
      <Form.InputNumber field="max" label="Max" />
      <Form.InputNumber field="decimals" label="Decimals" />
    </Form>
  );
};

const ThresholdsFieldConfigForm: React.FC = () => {
  const { panel, modifyPanel } = useContext(PanelEditContext);
  const formApi = useRef<any>();
  const maxStepValue = useRef<number>(0);
  const STEP = 10;
  const thresholds: Thresholds = get(panel, 'fieldConfig.defaults.thresholds', {});

  useEffect(() => {
    let i = 0;
    maxStepValue.current =
      get(
        maxBy(thresholds.steps, (o) => {
          i++;
          if (!has(o, '_key')) {
            set(o, '_key', `${i}`);
          }
          return o.value;
        }),
        'value',
        0
      ) || 0;
    if (formApi.current) {
      if (thresholds.steps) {
        thresholds.steps = orderBy(thresholds.steps, [(obj) => isNil(obj.value), 'value'], ['asc', 'desc']);
      }
      formApi.current.setValues(thresholds, { isOverride: true });
    }
  }, [formApi, thresholds]);

  return (
    <Form
      allowEmpty
      className="linsight-form linsight-panel-setting"
      getFormApi={(api: any) => (formApi.current = api)}
      extraTextPosition="middle"
      onValueChange={(values: Thresholds) => {
        modifyPanel({
          fieldConfig: {
            defaults: {
              thresholds: ObjectKit.removeUnderscoreProperties(values),
            },
          },
        });
      }}>
      <ArrayField field="steps">
        {({ arrayFields, addWithInitValue }) => {
          return (
            <>
              <Button
                icon={<IconPlusStroked />}
                onClick={() =>
                  addWithInitValue({
                    value: maxStepValue.current + STEP,
                    color: ColorKit.getColor(size(thresholds.steps)),
                  })
                }>
                Add threshold
              </Button>
              {arrayFields.map(({ field, key, remove }, i) => {
                const step: Threshold = get(thresholds, `steps.[${i}]`, {});
                if (isEmpty(step)) {
                  return null;
                }
                const fieldName = `${field}[value]`;
                return (
                  <Form.InputNumber
                    field={fieldName}
                    key={get(step, '_key', key)} // use step._key keep current field focus after sort
                    disabled={!step.value}
                    placeholder="Base"
                    noLabel
                    hideButtons={!step.value}
                    suffix={step.value && <IconDeleteStroked style={{ cursor: 'pointer' }} onClick={remove} />}
                    prefix={
                      <>
                        <Avatar
                          size="extra-extra-small"
                          style={{
                            margin: 8,
                            marginRight: 8,
                            width: 16,
                            height: 16,
                            borderRadius: '50%',
                            backgroundColor: `${step.color}`,
                          }}
                        />
                        {thresholds.mode === 'percentage' && (
                          <span style={{ marginRight: 6, fontSize: 12, color: 'var(--semi-color-text-2)' }}>%</span>
                        )}
                      </>
                    }
                  />
                );
              })}
            </>
          );
        }}
      </ArrayField>
      <Form.RadioGroup
        label="Thresholds mode"
        field="mode"
        type="button"
        extraText="Percentage means thresholds relative to min & max">
        <Radio value="absolute">Absolute</Radio>
        <Radio value="percentage">Percentage</Radio>
      </Form.RadioGroup>
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
      <Collapse
        expandIconPosition="left"
        defaultActiveKey={['chartOptions', 'panelOptions', 'visualizations', 'standardOptions', 'thresholds']}>
        <Collapse.Panel header="Chart repo. options" itemKey="chartOptions">
          <ChartRepoOptionsForm />
        </Collapse.Panel>
        <Collapse.Panel header="Panel options" itemKey="panelOptions">
          <PanelOptionsForm />
        </Collapse.Panel>
        <Collapse.Panel header="Visualizations" itemKey="visualizations">
          <VisualizationsForm />
        </Collapse.Panel>
        {OptionsEditor && (
          <OptionsEditor
            panel={ObjectKit.merge(VisualizationRepositoryInst.getDefaultOptions(`${panel.type}`), panel)}
            onOptionsChange={(options: PanelSetting) => {
              modifyPanel(options);
            }}
          />
        )}
        <Collapse.Panel header="Standard options" itemKey="standardOptions">
          <StandardOptionsForm />
        </Collapse.Panel>
        <Collapse.Panel header="Thresholds" itemKey="thresholds">
          <ThresholdsFieldConfigForm />
        </Collapse.Panel>
      </Collapse>
    </>
  );
};

export default PanelSetting;
