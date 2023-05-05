import { makeAutoObservable } from 'mobx';
import * as _ from 'lodash-es';
import { FormatKit } from '@src/utils';

class URLStore {
  public changed: boolean = false;
  public forceChanged: boolean = false;
  public path: string = '';

  private params: URLSearchParams = new URLSearchParams();
  private paramObj = {};
  private defaultParams = {}; // default params just save in store, don't put them into url params.

  constructor() {
    makeAutoObservable(this);

    this.applyURLChange();
  }

  private getPath(): string {
    const { hash } = window.location;
    let pathname = hash;
    if (hash.startsWith('#')) {
      pathname = hash.substring(1, hash.length);
    }
    if (pathname.indexOf('?') > -1) {
      pathname = pathname.split('?')[0];
    }
    return pathname;
  }

  changeURLParams(p: {
    path?: string;
    params?: { [key: string]: any };
    defaultParams?: { [key: string]: any };
    needDelete?: string[];
    clearAll?: boolean;
    clearTime?: boolean;
    forceChange?: boolean;
  }): void {
    const { params, defaultParams, needDelete, clearAll, clearTime, path, forceChange } = p;
    const oldSearchParams = this.getSearchParams();
    const searchParams = clearAll ? new URLSearchParams() : this.getSearchParams();
    const pathname = this.getPath();

    if (!clearAll) {
      (needDelete || []).map((key) => {
        searchParams.delete(key);
      });
    } else {
      if (!clearTime) {
        if (oldSearchParams.has('from')) {
          searchParams.set('from', oldSearchParams.get('from')!);
        }
        if (oldSearchParams.has('to')) {
          searchParams.set('to', oldSearchParams.get('to')!);
        }
      }
    }

    if (!_.isEmpty(defaultParams)) {
      this.changeDefaultParams(defaultParams || {}, false);
    }
    this.updateSearchParams(searchParams, params || {});
    // Because of Hash history cannot PUSH the same path so delete the logic of checking path consistency
    const paramsStr = searchParams.toString();
    if (oldSearchParams.toString() !== paramsStr || path) {
      console.log(
        'jjjjjjj.......',
        paramsStr,
        pathname,
        path,
        `${path ? path : pathname}${paramsStr && `?${paramsStr}`}`
      );
      window.history.replaceState(
        null,
        '',
        `${path ? `#${path}` : `#${pathname}`}${`${paramsStr}` && `?${paramsStr}`}`
      );
    }

    if (forceChange) {
      this.forceChange();
    }
  }

  public forceChange() {
    this.forceChanged = !this.forceChanged;
  }

  public changeDefaultParams(defaultParams: {}, change = true) {
    this.defaultParams = _.merge(_.cloneDeep(this.defaultParams), defaultParams);
    if (change) {
      this.changed = !this.changed;
    }
  }

  applyURLChange(): void {
    const groupParamsByKey = (params: any) =>
      [...params.entries()].reduce((acc, tuple) => {
        // getting the key and value from each tuple
        const [key, val] = tuple;
        const v = FormatKit.toObject(val);
        if (acc.hasOwnProperty(key)) {
          // if the current key is already an array, we'll add the value to it
          if (Array.isArray(acc[key])) {
            acc[key] = [...acc[key], v];
          } else {
            // if it's not an array, but contains a value, we'll convert it into an array
            // and add the current value to it
            acc[key] = [acc[key], v];
          }
        } else {
          // plain assignment if no special case is present
          acc[key] = v;
        }

        return acc;
      }, {});

    this.params = this.getSearchParams();
    this.paramObj = groupParamsByKey(this.params);
    const newPath = this.getPath();
    if (newPath != this.path) {
      // if change path need clear default params
      this.defaultParams = {};
    }
    this.path = newPath;
    this.changed = !this.changed;
  }

  public getParams(): object {
    return _.merge(_.cloneDeep(this.defaultParams), this.paramObj);
  }

  private getSearchParams(): URLSearchParams {
    if (window.location.href.indexOf('?') > -1) {
      return new URLSearchParams(window.location.href.split('?')[1]);
    } else {
      return new URLSearchParams();
    }
  }

  private updateSearchParams(searchParams: URLSearchParams, params: { [key: string]: any }) {
    _.forIn(params, (v, k) => {
      if (k) {
        if (!_.isUndefined(v)) {
          if (Array.isArray(v)) {
            searchParams.delete(k);
            v.forEach((oneValue) => searchParams.append(k, _.isString(oneValue) ? oneValue : `${oneValue}`));
          } else {
            searchParams.set(k, _.isString(v) ? v : `${v}`);
          }
        } else {
          searchParams.delete(k);
        }
      }
    });
  }
}

export default new URLStore();
