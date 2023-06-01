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

import { DraggableStateSnapshot } from 'react-beautiful-dnd';

const getDraggedDom = (draggableId: string): any => {
  return document.getElementById(draggableId);
};

const getDragPlaceholderProps = (event: any) => {
  const draggedDom = getDraggedDom(event.draggableId);

  if (!draggedDom) {
    return {};
  }

  const { clientHeight, clientWidth } = draggedDom;
  const sourceIndex = event.source.index;
  var clientY =
    parseFloat(window.getComputedStyle(draggedDom.parentNode).paddingTop) +
    [...draggedDom.parentNode.children].slice(0, sourceIndex).reduce((total, curr) => {
      const style = curr.currentStyle || window.getComputedStyle(curr);
      const marginBottom = parseFloat(style.marginBottom);
      return total + curr.clientHeight + marginBottom;
    }, 0);

  return {
    clientHeight,
    clientWidth,
    clientY,
    clientX: parseFloat(window.getComputedStyle(draggedDom.parentNode).paddingLeft),
  };
};

const getDragOverPlaceholderProps = (event: any) => {
  if (!event.destination) {
    return {};
  }

  const draggedDOM = getDraggedDom(event.draggableId);

  if (!draggedDOM) {
    return {};
  }

  const { clientHeight, clientWidth } = draggedDOM;
  const destinationIndex = event.destination.index;
  const sourceIndex = event.source.index;

  const childrenArray = [...draggedDOM.parentNode.children];
  const movedItem = childrenArray[sourceIndex];
  childrenArray.splice(sourceIndex, 1);

  const updatedArray = [
    ...childrenArray.slice(0, destinationIndex),
    movedItem,
    ...childrenArray.slice(destinationIndex + 1),
  ];

  var clientY =
    parseFloat(window.getComputedStyle(draggedDOM.parentNode).paddingTop) +
    updatedArray.slice(0, destinationIndex).reduce((total, curr) => {
      const style = curr.currentStyle || window.getComputedStyle(curr);
      const marginBottom = parseFloat(style.marginBottom);
      return total + curr.clientHeight + marginBottom;
    }, 0);

  return {
    clientHeight,
    clientWidth,
    clientY,
    clientX: parseFloat(window.getComputedStyle(draggedDOM.parentNode).paddingLeft),
  };
};

// https://github.com/atlassian/react-beautiful-dnd/blob/master/docs/guides/drop-animation.md#skipping-the-drop-animation
const getDraggbleItemStyle = (style: any, snapshot: DraggableStateSnapshot) => {
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

export default {
  getDragPlaceholderProps,
  getDragOverPlaceholderProps,
  getDraggbleItemStyle,
};
