import { makeAutoObservable } from 'mobx';

class MenuStore {
  private menus: Map<string, any> = new Map<string, any>();
  public currentMenu: any = null;

  constructor() {
    makeAutoObservable(this);
  }

  setCurrentMenu(path: string) {
    this.currentMenu = this.menus.get(path);
  }

  setMenus(menus: any) {
    this.menus.clear();
    menus.forEach((item: any) => {
      if (item.children) {
        item.children.forEach((child: any) => {
          if (child.path) {
            this.menus.set(child.path, child);
          }
        });
      }
      if (item.path) {
        this.menus.set(item.path, item);
      }
    });
  }
}

export default new MenuStore();
