import { MutableRefObject, useCallback, useEffect, useRef, useState } from 'react';
import { URLStore } from '@src/stores';
import { reaction } from 'mobx';
import * as _ from 'lodash-es';
import { useLocation } from 'react-router-dom';

const getParamValues = (keys: string[]) => {
  let newParams: any = URLStore.getParams();
  if (!_.isEmpty(keys)) {
    newParams = _.pick(newParams, keys);
  }
  return newParams || {};
};

function useParams(keys?: string[]) {
  const previous = useRef({}) as MutableRefObject<any>;
  const [params, setParams] = useState<any>(() => {
    const values = getParamValues(keys || []);
    previous.current = values;
    return values;
  });

  const buildParams = useCallback(() => {
    const values = getParamValues(keys || []);
    if (!_.isEqual(values, previous.current)) {
      previous.current = values;
      setParams(values);
    }
    // NOTE: Don't add keys into deps
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    console.log('ssss');
  }, []);

  // NOTE: don't use {...params}
  return params;
}

function useNavigate() {
  return useCallback(
    (
      path: string,
      p?: {
        params?: { [key: string]: any };
        defaultParams?: { [key: string]: any };
        needDelete?: string[];
        clearAll?: boolean;
        clearTime?: boolean;
        forceChange?: boolean;
      }
    ) => {
      URLStore.changeURLParams({ path: path, ...p });
    },
    []
  );
}

export { useParams, useNavigate };
