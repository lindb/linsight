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
import React, { createContext, MutableRefObject, useEffect, useMemo, useRef, useState } from 'react';
import { Query, Tracker } from '@src/types';
import { StringKit } from '@src/utils';
import { pullAt, forIn, set } from 'lodash-es';

export const TargetsContext = createContext({
  targets: [] as Query[],
  activeIds: [] as string[],
  isActive: (_refId: string): boolean => {
    return false;
  },
  toggleActiveRefId: (_refId: string) => {},
  toggleTargetHide: (_index: number) => {},
  swapTargets: (_sourceIndex: number, _destIndex: number) => {},
  addTarget: (_newTarget: Query) => {},
  deleteTarget: (_index: number) => {},
  updateTargetConfig: (_index: number, _cfg: Query) => {},
});

const generateRefId = (existIds: Set<string>): string => {
  for (let i = 0; i < 26; i++) {
    const refId = StringKit.generateCharSeq(i);
    if (!existIds.has(refId)) {
      return refId;
    }
  }
  return '';
};

const checkAndSetTargetRefId = (targets: Query[]): Query[] => {
  const ids = new Set<string>();
  (targets || []).forEach((q: Query) => {
    if (!q.refId) {
      q.refId = generateRefId(ids);
    }
    ids.add(q.refId);
  });
  return targets;
};

export const TargetsContextProvider: React.FC<{
  initTargets: Query[];
  onTargetsChange: (targets: Query[]) => void;
  children: React.ReactNode;
}> = (props) => {
  const { initTargets, onTargetsChange, children } = props;
  const targetsAfterCheck = checkAndSetTargetRefId(initTargets);
  const refIds = useRef() as MutableRefObject<Set<string>>;
  const activeRefIds = useRef() as MutableRefObject<Set<string>>;
  const targetsTracker = useRef() as MutableRefObject<Tracker<Query[]>>;
  const [targets, setTargets] = useState<Query[]>(targetsAfterCheck);
  const [activeIds, setActiveIds] = useState<string[]>(() => {
    return (targetsAfterCheck || []).map((q: Query) => {
      return `${q.refId}`;
    });
  });

  const genRefId = (): string => {
    return generateRefId(refIds.current);
  };

  useEffect(() => {
    if (targetsTracker.current.isChanged(targets)) {
      targetsTracker.current.setNewVal(targets);
      onTargetsChange(targets);
    }
  }, [targets, onTargetsChange]);

  useMemo(() => {
    refIds.current = new Set<string>();
    activeRefIds.current = new Set<string>();
    targetsTracker.current = new Tracker(targetsAfterCheck);
    targetsTracker.current.setNewVal(targetsAfterCheck);
    (targetsAfterCheck || []).forEach((q: Query) => {
      const refId = `${q.refId}`;
      activeRefIds.current.add(refId);
      refIds.current.add(refId);
    });
  }, [targetsAfterCheck]);

  const isActive = (refId: string): boolean => {
    return activeRefIds.current.has(refId);
  };

  const toggleActiveRefId = (refId: string) => {
    if (activeRefIds.current.has(refId)) {
      activeRefIds.current.delete(refId);
    } else {
      activeRefIds.current.add(refId);
    }
    setActiveIds(Array.from(activeRefIds.current));
  };

  const updateTargetConfig = (index: number, cfg: Query) => {
    const target = targets[index];
    forIn(cfg, function (value, key) {
      set(target, key, value);
    });
    setTargets([...targets]);
  };

  const toggleTargetHide = (index: number) => {
    targets[index].hide = !targets[index].hide;
    setTargets([...targets]);
  };

  const deleteTarget = (index: number) => {
    const deleted = pullAt(targets, index);
    (deleted || []).forEach((target: Query) => {
      refIds.current.delete(`${target.refId}`);
      activeRefIds.current.add(`${target.refId}`);
    });
    setActiveIds(Array.from(activeRefIds.current));
    setTargets([...targets]);
  };

  const addTarget = (newTarget: Query) => {
    const newRefId = genRefId();
    newTarget.refId = newRefId;
    refIds.current.add(`${newTarget.refId}`);
    activeRefIds.current.add(`${newTarget.refId}`);
    targets.push(newTarget);
    setTargets([...targets]);
    setActiveIds(Array.from(activeRefIds.current));
  };

  const swapTargets = (sourceIndex: number, destIndex: number) => {
    const [removed] = targets.splice(sourceIndex, 1);
    targets.splice(destIndex, 0, removed);
    setTargets([...targets]);
  };

  return (
    <TargetsContext.Provider
      value={{
        targets,
        activeIds,
        isActive,
        toggleActiveRefId,
        toggleTargetHide,
        swapTargets,
        addTarget,
        deleteTarget,
        updateTargetConfig,
      }}>
      {children}
    </TargetsContext.Provider>
  );
};
