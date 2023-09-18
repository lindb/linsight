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
import { Form } from '@douyinfe/semi-ui';
import { QueryEditContext } from '@src/contexts';
import { Query, QueryEditorProps, Tracker } from '@src/types';
import { get } from 'lodash-es';
import React, { MutableRefObject, useCallback, useContext, useEffect, useMemo, useRef } from 'react';
import './query-edit.scss';

const QueryEditor: React.FC<QueryEditorProps> = (props) => {
  const { datasource } = props;
  const { target, modifyTarget } = useContext(QueryEditContext);
  const requestTracker = useRef() as MutableRefObject<Tracker<object>>;
  const formApi = useRef() as MutableRefObject<any>;

  const getInitRequest = useCallback(() => {
    return get(target, 'request', {});
  }, [target]);

  useMemo(() => {
    requestTracker.current = new Tracker(getInitRequest());
  }, [getInitRequest]);

  useEffect(() => {
    formApi.current.setValues(getInitRequest());
  }, [datasource, getInitRequest]);

  return (
    <>
      <Form
        getFormApi={(api: any) => {
          formApi.current = api;
        }}
        className="lingo-query-editor"
        layout="horizontal"
        labelPosition="left"
        onSubmit={(values: any) => {
          if (!requestTracker.current.isChanged(values)) {
            return;
          }
          requestTracker.current.setNewVal(values);
          // change query edit context's values
          modifyTarget({ request: values } as Query);
        }}>
        <Form.Input label="TraceId" field="traceId" style={{ width: 400 }} />
      </Form>
    </>
  );
};

export default QueryEditor;
