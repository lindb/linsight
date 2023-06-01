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
import React, { useState } from 'react';
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
import { cloneDeep, get, isEmpty } from 'lodash-es';
import { QueryEditorStore } from '@src/stores';
import { observer } from 'mobx-react-lite';
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
import { toJS } from 'mobx';
import classNames from 'classnames';
import { DNDKit } from '@src/utils';

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

const Targets: React.FC<{ datasource: DatasourceInstance }> = observer((props) => {
  const { datasource } = props;
  return (
    <Collapse activeKey={QueryEditorStore.getActiveRefIds()} expandIconPosition="left" clickHeaderToExpand={false}>
      {toJS(QueryEditorStore.targets).map((target: Query, index: number) => {
        // NOTE: if not ues toJS, mobx observer is not working
        const refId = target.refId;
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
                    // style={{ backgroundColor: snapshot.isDragging ? 'red' : 'unset' }}
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
                          <IconHandle className="drag" size="large" {...provided.dragHandleProps} />
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
                </div>
              );
            }}
          </Draggable>
        );
      })}
    </Collapse>
  );
});

const QueryEditor: React.FC<{ datasource: DatasourceInstance }> = (props) => {
  const { datasource } = props;
  const plugin = datasource.plugin;
  const [placeholderProps, setPlaceholderProps] = useState<any>({});
  const QueryEditor = plugin.components.QueryEditor;
  if (!QueryEditor) {
    return <></>;
  }

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

          QueryEditorStore.swapTargets(source.index, destination.index);
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

export default QueryEditor;
