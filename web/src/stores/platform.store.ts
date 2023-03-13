import { MouseEvent } from '@src/types';
import { makeAutoObservable } from 'mobx';

class PlatformStore {
  public mouseEvent: MouseEvent | null = null;

  constructor() {
    makeAutoObservable(this);
  }

  setMouseEvent(e: MouseEvent) {
    this.mouseEvent = e;
  }
}

export default new PlatformStore();

