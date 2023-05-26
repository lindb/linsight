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
import React, { useMemo, useState, forwardRef, useRef, useEffect } from 'react';
import { DashboardStore } from '@src/stores';
import { get, isEmpty, cloneDeep } from 'lodash-es';
import { Button, Table, Typography } from '@douyinfe/semi-ui';
import { IconPlusStroked, IconDeleteStroked, IconCopyStroked, IconEyeOpened } from '@douyinfe/semi-icons';
import { StatusTip } from '@src/components';
import { observer } from 'mobx-react-lite';
import { useSearchParams } from 'react-router-dom';
import ViewVariables from './ViewVariables';
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
import './variables.scss';
import { Variable, VariableHideType } from '@src/types';
const { Text } = Typography;

// https://github.com/atlassian/react-beautiful-dnd/blob/master/docs/guides/drop-animation.md#skipping-the-drop-animation
const getStyle = (style: any, snapshot: DraggableStateSnapshot) => {
  if (!snapshot.isDropAnimating) {
    return {
      ...style,
    };
  }
  return {
    ...style,
    // cannot be 0, but make it super tiny
    transitionDuration: `0.001s`,
  };
};

const TableBody = ({ ...props }) => {
  return (
    <Droppable droppableId="table" direction="vertical">
      {(provided: DroppableProvided, _snapshot: DroppableStateSnapshot) => (
        <tbody ref={provided.innerRef} {...props}></tbody>
      )}
    </Droppable>
  );
};

const TableRow = (props: any) => {
  const { index } = props;
  return (
    <Draggable key={index} draggableId={`${index}`} index={index}>
      {(provided: DraggableProvided, snapshot: DraggableStateSnapshot) => {
        return (
          <tr
            ref={provided.innerRef}
            {...props}
            className={classNames('semi-table-row', { 'row-dragging': snapshot.isDragging })}
            {...provided.draggableProps}
            {...provided.dragHandleProps}
            style={getStyle(provided.draggableProps.style, snapshot)}
          />
        );
      }}
    </Draggable>
  );
};

// https://github.com/atlassian/react-beautiful-dnd/blob/master/docs/patterns/tables.md
const TableCell = (props: any) => {
  const [snapshot, setSnapshot] = useState<null | DOMRect>(null);
  const componentJustMounted = useRef(true);
  const ref = useRef<null | HTMLElement>(null);
  useEffect(() => {
    if (componentJustMounted.current) {
      if (ref.current) {
        // save table cell width after mounted
        setSnapshot(ref.current.getBoundingClientRect());
      }
    }
    componentJustMounted.current = false;
  }, []);

  if (snapshot) {
    return <td {...props} style={{ width: `${snapshot.width}px` }} />;
  }
  return <td ref={ref} {...props} />;
};

const VariableList: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const variables: Variable[] = DashboardStore.getVariables();
  const [preview, setPreview] = useState(false);

  const components = useMemo(() => {
    return {
      body: {
        wrapper: (val: any) => <TableBody {...val} />,
        row: forwardRef(function Row(val: any, _ref: any) {
          return <TableRow {...val} />;
        }),
        cell: forwardRef(function Cell(val: any, _ref: any) {
          return <TableCell {...val} />;
        }),
      },
    };
  }, []);

  return (
    <>
      <div style={{ display: 'flex', flexDirection: 'row', gap: 4 }}>
        <Button
          icon={<IconPlusStroked />}
          onClick={() => {
            const len = variables.length;
            const variable = `variable${len}`;
            DashboardStore.addVariable({ name: variable, label: 'Variable', hide: VariableHideType.LabelAndValue });
            searchParams.set('edit', `${len}`);
            setSearchParams(searchParams);
          }}>
          New
        </Button>
        <Button
          type="secondary"
          icon={<IconEyeOpened />}
          onClick={() => {
            setPreview(!preview);
          }}>
          Preview
        </Button>
      </div>
      {preview && (
        <div style={{ marginTop: 12 }}>
          <ViewVariables />
        </div>
      )}
      <DragDropContext
        onDragEnd={(result: any) => {
          const { destination, source, reason } = result;
          // nothing to do
          if (!destination || reason === 'CANCEL') {
            return;
          }
          // same
          if (destination.droppableId === source.droppableId && destination.index === source.index) {
            return;
          }

          DashboardStore.reorderVariables(source.index, destination.index);
        }}>
        <Table
          className="lin-variables-table"
          size="small"
          pagination={false}
          bordered
          empty={<StatusTip isEmpty={isEmpty(variables)} />}
          dataSource={variables}
          components={components as any}
          onRow={(record, index) => ({
            index,
            record,
          })}
          columns={[
            {
              title: 'Name',
              dataIndex: 'name',
              render: (_text: any, r: Variable, index: number) => {
                return (
                  <Text
                    style={{ padding: '12px 0 12px 0' }}
                    link
                    onClick={() => {
                      searchParams.set('edit', `${index}`);
                      setSearchParams(searchParams);
                    }}>
                    {r.name}
                  </Text>
                );
              },
            },
            { title: 'Label', dataIndex: 'label' },
            { title: 'Definition' },
            {
              title: 'Operations',
              width: 120,
              render: (_text: any, r: Variable, index: number) => {
                return (
                  <div style={{ display: 'flex', flexDirection: 'row', gap: 4 }}>
                    <Button
                      type="primary"
                      icon={<IconCopyStroked size="large" />}
                      onClick={() => {
                        const newVariable = cloneDeep(r);
                        newVariable.name = `copy_of_${newVariable.name}`;
                        DashboardStore.addVariable(newVariable);
                      }}
                    />
                    <Button
                      type="danger"
                      icon={<IconDeleteStroked size="large" />}
                      onClick={() => DashboardStore.deleteVariable(`${index}`)}
                    />
                  </div>
                );
              },
            },
          ]}
        />
      </DragDropContext>
    </>
  );
};

export default observer(VariableList);
