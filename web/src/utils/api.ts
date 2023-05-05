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
import axios from 'axios';
import qs from 'qs';
import * as _ from 'lodash-es';

axios.defaults.baseURL = '/api/v1';
axios.defaults.timeout = 60000; // set timeout
axios.defaults.headers.common['Content-Type'] = 'application/json';

async function GET<T>(url: string, params?: { [index: string]: any } | undefined): Promise<T> {
  const target = url + (params ? `?${qs.stringify(params, { arrayFormat: 'repeat' })}` : '');
  return axios
    .get<T>(target)
    .then((result) => {
      return Promise.resolve(result.data);
    })
    .catch((err) => {
      return Promise.reject(err);
    });
}

async function PUT<T>(url: string, params?: { [index: string]: any } | undefined): Promise<T> {
  return axios
    .put<T>(url, params)
    .then((result) => {
      return Promise.resolve(result.data);
    })
    .catch((err) => {
      return Promise.reject(err);
    });
}

async function POST<T>(url: string, params?: { [index: string]: any } | undefined): Promise<T> {
  return axios
    .post<T>(url, params)
    .then((result) => {
      return Promise.resolve(result.data);
    })
    .catch((err) => {
      return Promise.reject(err);
    });
}

async function DELETE<T>(url: string): Promise<T> {
  return axios
    .delete<T>(url)
    .then((result) => {
      return Promise.resolve(result.data);
    })
    .catch((err) => {
      return Promise.reject(err);
    });
}

const getErrorMsg = (err: any) => {
  return _.get(err, 'response.data', 'Unknown internal error');
};

const getErrorCode = (err: any) => {
  return _.get(err, 'response.status', 0);
};

export default {
  getErrorMsg,
  getErrorCode,
  GET,
  PUT,
  POST,
  DELETE,
};
