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
import React from 'react';
import { useRouteError } from 'react-router-dom';
import { toString, get, isEmpty } from 'lodash-es';
import { Empty, Typography } from '@douyinfe/semi-ui';
const { Text } = Typography;
import ErrorImg from '@src/images/error2.svg';

const ErrorPage: React.FC = () => {
  const error = useRouteError();
  const stack = get(error, 'stack');
  return (
    <Empty
      title="Oops! Something went wrong"
      style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}
      image={<img src={ErrorImg} style={{ width: 250, height: 250 }} />}
      darkModeImage={<img src={ErrorImg} style={{ width: 250, height: 250 }} />}
      layout="horizontal"
      description={<Text type="danger">{toString(error)}</Text>}>
      {!isEmpty(stack) && <pre>{get(error, 'stack')}</pre>}
    </Empty>
  );
};

export default ErrorPage;
