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
import { Layout } from '@douyinfe/semi-ui';
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
  DroppableProvided,
  DroppableStateSnapshot,
  DraggableProvided,
  DraggableStateSnapshot,
} from 'react-beautiful-dnd';
import * as _ from 'lodash-es';

const { Header } = Layout;

const dashboardValue = {
  rows: [
    {
      title: 'row-1',
      panels: [
        { title: 'title-1' },
        { title: 'title-2' },
        { title: 'title-3' },
        { title: 'title-4' },
        { title: 'title-5' },
      ],
    },
    {
      title: 'row-2',
      panels: [
        { title: 'title-6' },
        { title: 'title-7' },
        { title: 'title-8' },
        { title: 'title-9' },
        { title: 'title-10' },
      ],
    },
    {
      title: 'row-3',
      panels: [
        { title: 'title-11' },
        { title: 'title-12' },
        { title: 'title-13' },
        { title: 'title-14' },
        { title: 'title-15' },
      ],
    },
  ],
};

const grid = 8;

const getItemStyle = (isDragging: any, draggableStyle: any) => ({
  // some basic styles to make the items look a bit nicer
  userSelect: 'none',
  padding: grid * 2,

  // change background colour if dragging
  background: isDragging ? 'lightgreen' : 'grey',

  // styles we need to apply on draggables
  ...draggableStyle,
});

const getListStyle = (isDraggingOver: any) => ({
  background: isDraggingOver ? 'lightblue' : 'lightgrey',
  padding: grid,
});

const PanelList: React.FC<{
  id: string;
  panels: any;
}> = (props) => {
  const { id, panels } = props;
  return (
    <Droppable type="panel" droppableId={id} direction="horizontal">
      {(dropProvided: DroppableProvided, dropSnapshot: DroppableStateSnapshot) => (
        <div
          className="dashboard-panel-list"
          ref={dropProvided.innerRef}
          {...dropProvided.droppableProps}
          style={getListStyle(dropSnapshot.isDraggingOver)}>
          {panels.map((item: any, index: number) => (
            <Draggable key={item.title} draggableId={item.title} index={index}>
              {(dragProvided: DraggableProvided, dragSnapshot: DraggableStateSnapshot) => (
                <div
                  className="dashboard-panel"
                  ref={dragProvided.innerRef}
                  {...dragProvided.draggableProps}
                  {...dragProvided.dragHandleProps}
                  style={getItemStyle(dragSnapshot.isDragging, dragProvided.draggableProps.style)}>
                  {item.title}
                </div>
              )}
            </Draggable>
          ))}
          {dropProvided.placeholder}
        </div>
      )}
    </Droppable>
  );
};

const DashboardRow: React.FC<{
  index: number;
  row: any;
}> = (props) => {
  const { index, row } = props;
  return (
    <Draggable draggableId={row.title} index={index}>
      {(dragProvided: DraggableProvided, dragSnapshot: DraggableStateSnapshot) => (
        <div
          className="dashboard-row"
          ref={dragProvided.innerRef}
          {...dragProvided.draggableProps}
          {...dragProvided.dragHandleProps}
          style={getItemStyle(dragSnapshot.isDragging, dragProvided.draggableProps.style)}>
          <div>{row.title}</div>
          <div>
            <PanelList id={row.title} panels={row.panels} />
          </div>
        </div>
      )}
    </Draggable>
  );
};

const EditDashboard: React.FC = () => {
  const [dashboard, setDashboard] = useState(dashboardValue);

  const onDragEnd = (result: DropResult) => {
    if (!result.destination) {
      // dropped outside the list
      return;
    }
    const sourceIdx = result.source.index;
    const destIdx = result.destination.index;
    if (result.type === 'panel') {
      const sourceRow = _.find(dashboard.rows, { title: result.source.droppableId });
      const destRow = _.find(dashboard.rows, { title: result.destination.droppableId });
      const removed = sourceRow?.panels.splice(sourceIdx, 1) as any;
      destRow?.panels.splice(destIdx, 0, ...removed);
    } else if (result.type === 'row') {
      const [removed] = dashboard.rows.splice(sourceIdx, 1);
      dashboard.rows.splice(destIdx, 0, removed);
    }
    console.log(dashboard);
  };

  return (
    <>
      <DragDropContext onDragEnd={onDragEnd}>
        <Droppable droppableId="dashboard" type="row" direction="vertical">
          {(dropProvided: DroppableProvided) => (
            <div ref={dropProvided.innerRef} {...dropProvided.droppableProps}>
              {dashboard.rows.map((item, index) => (
                <DashboardRow key={item.title} row={item} index={index} />
              ))}
              {dropProvided.placeholder}
            </div>
          )}
        </Droppable>
      </DragDropContext>
    </>
  );
};

export default EditDashboard;
