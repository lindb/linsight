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
import React from 'react';
import { Button, Collapse, Tag, Typography } from '@douyinfe/semi-ui';
import {
  IconCopy,
  IconHandle,
  IconPlusStroked,
  IconDeleteStroked,
  IconChevronDown,
  IconChevronUp,
} from '@douyinfe/semi-icons';
import { QueryEditContextProvider } from '@src/contexts';
import { DatasourceInstance, Query } from '@src/types';
import { cloneDeep, get } from 'lodash-es';
import { QueryEditorStore } from '@src/stores';
import { observer } from 'mobx-react-lite';
import Icon from '../common/Icon';
import './query.editor.scss';

const { Text } = Typography;

const MetricQueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;
  const QueryEditor = plugin.components.QueryEditor;

  if (!QueryEditor) {
    return null;
  }
  return <QueryEditor datasource={datasource} />;
};

const QueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;

  const QueryEditor = plugin.components.QueryEditor;
  if (!QueryEditor) {
    return <></>;
  }
  return (
    <div className="query-editor">
      <Collapse activeKey={QueryEditorStore.getActiveRefIds()} expandIconPosition="left" clickHeaderToExpand={false}>
        {QueryEditorStore.targets.map((target: Query, index: number) => {
          const refId = target.refId;
          return (
            <Collapse.Panel
              showArrow={false}
              itemKey={refId}
              key={refId}
              header={
                <div className="query-item">
                  <div
                    className="query-desc"
                    onClick={() => {
                      QueryEditorStore.toggleActiveRefId(refId);
                    }}>
                    <Button
                      icon={QueryEditorStore.isActive(refId) ? <IconChevronDown /> : <IconChevronUp />}
                      size="small"
                      theme="borderless"
                      type="tertiary"
                    />
                    <Text type="tertiary">{refId}</Text>
                    <Text type="tertiary" size="small">
                      ({datasource.setting.name})
                    </Text>
                    {target.hide && (
                      <Tag size="small" color="orange">
                        Disabled
                      </Tag>
                    )}
                  </div>
                  <div className="actions">
                    <Button
                      size="small"
                      theme="borderless"
                      type="tertiary"
                      icon={<IconCopy />}
                      onClick={() => {
                        const newTarget = cloneDeep(target);
                        newTarget.refId = QueryEditorStore.genRefId();
                        QueryEditorStore.addTarget(newTarget);
                      }}
                    />
                    <Button
                      size="small"
                      theme="borderless"
                      style={{ color: target.hide ? 'var(--semi-color-primary)' : '' }}
                      type={target.hide ? 'primary' : 'tertiary'}
                      icon={<Icon icon={target.hide ? 'eye-close' : 'eye'} />}
                      onClick={() => {
                        QueryEditorStore.toggleTargetHide(index);
                      }}
                    />
                    <Button
                      size="small"
                      theme="borderless"
                      type="tertiary"
                      icon={<IconDeleteStroked />}
                      onClick={() => {
                        QueryEditorStore.deleteTarget(index);
                      }}
                    />
                    <IconHandle className="drag grid-drag-handle" size="large" />
                  </div>
                </div>
              }>
              <QueryEditContextProvider
                initValues={get(target, 'request', {})}
                onValuesChange={(values: object) => {
                  QueryEditorStore.updateTargetConfig(index, { request: values } as Query);
                }}>
                <MetricQueryEditor datasource={datasource} />
              </QueryEditContextProvider>
            </Collapse.Panel>
          );
        })}
      </Collapse>
      <Button
        style={{ marginTop: 12 }}
        icon={<IconPlusStroked />}
        onClick={() => {
          QueryEditorStore.addTarget({
            refId: QueryEditorStore.genRefId(),
            datasource: { uid: datasource.setting.uid, type: datasource.setting.type },
          } as Query);
        }}>
        Query
      </Button>
    </div>
  );
};

export default observer(QueryEditor);
