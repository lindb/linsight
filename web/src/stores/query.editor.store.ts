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

import { Query } from '@src/types';
import { makeAutoObservable } from 'mobx';
import { pullAt, cloneDeep, isEqual, forIn, set } from 'lodash-es';
import { StringKit } from '@src/utils';

class QueryEditorStore {
  private refIds: Set<string> = new Set();
  private activeRefIds: Set<string> = new Set();
  public targets: Query[] = [];
  constructor() {
    makeAutoObservable(this);
  }

  genRefId(): string {
    for (let i = 0; i < 26; i++) {
      const refId = StringKit.generateCharSeq(i);
      if (!this.refIds.has(refId)) {
        return refId;
      }
    }
    return '';
  }

  getActiveRefIds(): string[] {
    return Array.from(this.activeRefIds.keys());
  }

  isActive(refId: string): boolean {
    return this.activeRefIds.has(refId);
  }

  toggleActiveRefId(refId: string) {
    if (this.activeRefIds.has(refId)) {
      this.activeRefIds.delete(refId);
    } else {
      this.activeRefIds.add(refId);
    }
  }

  setTargets(targets: Query[]) {
    if (isEqual(this.targets, targets)) {
      // same targets, ignore it
      return;
    }
    this.targets = cloneDeep(targets || []);
    (targets || []).forEach((q: Query) => {
      if (!q.refId) {
        q.refId = this.genRefId();
      }
      this.activeRefIds.add(q.refId);
      this.refIds.add(q.refId);
    });
  }

  updateTargetConfig(index: number, cfg: Query) {
    const target = this.targets[index];
    forIn(cfg, function (value, key) {
      set(target, key, value);
    });
  }

  toggleTargetHide(index: number) {
    this.targets[index].hide = !this.targets[index].hide;
  }

  deleteTarget(index: number) {
    const deleted = pullAt(this.targets, index);
    (deleted || []).forEach((target: Query) => {
      this.refIds.delete(target.refId);
      this.activeRefIds.add(target.refId);
    });
  }

  addTarget(newTarget: Query) {
    this.refIds.add(newTarget.refId);
    this.activeRefIds.add(newTarget.refId);
    this.targets.push(newTarget);
  }
}

export default new QueryEditorStore();
