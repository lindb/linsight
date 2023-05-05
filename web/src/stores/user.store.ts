import { User } from '@src/types';
import { makeAutoObservable } from 'mobx';

class UserStore {
  user?: User;

  constructor() {
    makeAutoObservable(this);
  }

  setUser(user: User) {
    this.user = user;
  }
}

export default new UserStore();
