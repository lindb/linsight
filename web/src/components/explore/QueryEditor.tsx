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
import { Button, Collapse, Tag, Typography } from '@douyinfe/semi-ui';
import {
  IconCopy,
  IconHandle,
  IconPlusStroked,
  IconDeleteStroked,
  IconChevronDown,
  IconChevronRight,
} from '@douyinfe/semi-icons';
import { PanelEditContext, QueryEditContextProvider, TargetsContext, TargetsContextProvider } from '@src/contexts';
import { DatasourceInstance, Query } from '@src/types';
import { cloneDeep, get, isEmpty } from 'lodash-es';
import Icon from '../common/Icon';
import './query.editor.scss';
import {
  DragDropContext,
  Draggable,
  DraggableProvided,
  DraggableStateSnapshot,
  Droppable,
  DroppableProvided,
  DroppableStateSnapshot,
} from 'react-beautiful-dnd';
import classNames from 'classnames';
import { DNDKit } from '@src/utils';
import DatasourceSelectForm from '../input/DatasourceSelectForm';
import { MixedDatasource } from '@src/constants';
import { DatasourceStore } from '@src/stores';

const { Text } = Typography;

const Targets: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource: defaultDatasource /*default datasource*/ } = props;
  const {
    targets,
    activeIds,
    isActive,
    toggleActiveRefId,
    toggleTargetHide,
    addTarget,
    deleteTarget,
    updateTargetConfig,
  } = useContext(TargetsContext);

  return (
    <Collapse activeKey={activeIds} expandIconPosition="left" clickHeaderToExpand={false}>
      {targets.map((target: Query, index: number) => {
        const refId = `${target.refId}`;
        const datasourceUID = get(target, 'datasource.uid', get(defaultDatasource, 'setting.uid'));
        const datasource = DatasourceStore.getDatasource(`${datasourceUID}`);
        if (!datasource || datasource.setting.uid === MixedDatasource) {
          return null;
        }
        const QueryEditor = datasource.plugin.getQueryEditor();
        return (
          <Draggable key={refId} draggableId={refId} index={index}>
            {(provided: DraggableProvided, snapshot: DraggableStateSnapshot) => {
              return (
                <div
                  id={refId}
                  {...provided.draggableProps}
                  ref={provided.innerRef}
                  style={DNDKit.getDraggbleItemStyle(provided.draggableProps.style, snapshot)}
                  className={classNames({ 'query-item-dragging': snapshot.isDragging })}>
                  <Collapse.Panel
                    showArrow={false}
                    itemKey={refId}
                    header={
                      <div className="query-item">
                        <div className="query-desc">
                          <Button
                            icon={isActive(refId) ? <IconChevronDown /> : <IconChevronRight />}
                            size="small"
                            theme="borderless"
                            type="tertiary"
                            onClick={() => {
                              toggleActiveRefId(refId);
                            }}
                          />
                          <Text type="tertiary">{refId}</Text>
                          {defaultDatasource.setting.uid === MixedDatasource ? (
                            <DatasourceSelectForm
                              noLabel
                              value={datasourceUID}
                              style={{ width: 200 }}
                              onChange={(instance: DatasourceInstance) => {
                                updateTargetConfig(index, {
                                  datasource: { uid: instance.setting.uid, type: instance.setting.type },
                                } as any);
                              }}
                            />
                          ) : (
                            <Text type="tertiary" size="small">
                              ({defaultDatasource.setting.name})
                            </Text>
                          )}
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
                              addTarget(newTarget);
                            }}
                          />
                          <Button
                            size="small"
                            theme="borderless"
                            style={{ color: target.hide ? 'var(--semi-color-primary)' : '' }}
                            type={target.hide ? 'primary' : 'tertiary'}
                            icon={<Icon icon={target.hide ? 'eye-close' : 'eye'} />}
                            onClick={() => {
                              toggleTargetHide(index);
                            }}
                          />
                          <Button
                            size="small"
                            theme="borderless"
                            type="tertiary"
                            icon={<IconDeleteStroked />}
                            onClick={() => {
                              deleteTarget(index);
                            }}
                          />
                          <IconHandle className="drag" size="large" {...provided.dragHandleProps} />
                        </div>
                      </div>
                    }>
                    <QueryEditContextProvider
                      initTarget={target}
                      onTargetChange={(newTarget: Query) => {
                        updateTargetConfig(index, newTarget);
                      }}>
                      <QueryEditor datasource={datasource || defaultDatasource} />
                    </QueryEditContextProvider>
                  </Collapse.Panel>
                </div>
              );
            }}
          </Draggable>
        );
      })}
    </Collapse>
  );
};

const TargetsEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const [placeholderProps, setPlaceholderProps] = useState<any>({});
  const { swapTargets, addTarget } = useContext(TargetsContext);
  return (
    <div className="query-editor">
      <DragDropContext
        onDragStart={(event: any) => {
          setPlaceholderProps(DNDKit.getDragPlaceholderProps(event));
        }}
        onDragUpdate={(event: any) => {
          setPlaceholderProps(DNDKit.getDragOverPlaceholderProps(event));
        }}
        onDragEnd={(result: any) => {
          setPlaceholderProps({});
          const { destination, source, reason } = result;
          // nothing to do
          if (!destination || reason === 'CANCEL') {
            return;
          }
          // same
          if (destination.droppableId === source.droppableId && destination.index === source.index) {
            return;
          }
          swapTargets(source.index, destination.index);
        }}>
        <Droppable droppableId="query-editors" direction="vertical">
          {(provided: DroppableProvided, _snapshot: DroppableStateSnapshot) => {
            return (
              <div ref={provided.innerRef} {...provided.droppableProps} style={{ position: 'relative' }}>
                <Targets datasource={datasource} />
                {provided.placeholder}
                {!isEmpty(placeholderProps) && (
                  <div
                    className="placeholder"
                    style={{
                      top: placeholderProps.clientY,
                      left: placeholderProps.clientX,
                      height: placeholderProps.clientHeight,
                      width: placeholderProps.clientWidth,
                    }}
                  />
                )}
              </div>
            );
          }}
        </Droppable>
      </DragDropContext>
      <Button
        style={{ marginTop: 12 }}
        icon={<IconPlusStroked />}
        onClick={() => {
          let ds = datasource;
          // if panel's datasource type is mixed, need select default datasource
          if (ds.setting.uid === MixedDatasource) {
            ds = DatasourceStore.getDefaultDatasource() as any;
          }
          addTarget({
            datasource: { uid: ds.setting.uid, type: ds.setting.type },
          } as Query);
        }}>
        Query
      </Button>
    </div>
  );
};

const QueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const { panel, modifyPanel } = useContext(PanelEditContext);
  return (
    <TargetsContextProvider
      initTargets={get(panel, 'targets', [])}
      onTargetsChange={(targets) => {
        modifyPanel({ targets: targets });
      }}>
      <TargetsEditor datasource={datasource} />
    </TargetsContextProvider>
  );
};

export default QueryEditor;
