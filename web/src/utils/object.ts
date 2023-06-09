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
import { mergeWith, isArray, isString, transform, isObject, pickBy, isEmpty } from 'lodash-es';

const merge = (dest: object, src: object): any => {
  return mergeWith(dest, src, (objValue: any, srcValue: any) => {
    // if array, need overwrite with new value.
    if (isArray(objValue)) {
      return srcValue;
    }
  });
};

/**
 * Remove all underscore props
 */
const removeUnderscoreProperties = (obj: Record<string, any>): Record<string, any> => {
  return transform(obj, (result, value, key) => {
    let val = value;
    if (isObject(value) || isArray(value)) {
      val = removeUnderscoreProperties(value);
    }
    if (isString(key)) {
      // handle object
      if (!key.startsWith('_')) {
        result[key] = val;
      }
    } else {
      // handle array
      result[key] = val;
    }
  });
};

const cleanEmptyProperties = (obj: object): any => {
  return pickBy(obj, (value) => {
    if (isObject(value) && !isArray(value)) {
      return !isEmpty(cleanEmptyProperties(value));
    }
    return !isEmpty(value);
  });
};

export default {
  merge,
  removeUnderscoreProperties,
  cleanEmptyProperties,
};
