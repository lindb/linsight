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
import { Layout } from '@douyinfe/semi-ui';
const { Footer: UIFooter } = Layout;

const Footer: React.FC = () => {
  return (
    <UIFooter
      style={{
        textAlign: 'center',
        height: 48,
        paddingTop: 14,
        fontSize: 12,
        justifyContent: 'space-between',
        color: 'var(--semi-color-text-2)',
        backgroundColor: 'rgba(var(--semi-grey-0), 1)',
      }}>
      <span>Copyright &copy; {new Date().getFullYear()} LinDB Labs. All Rights Reserved.</span>
    </UIFooter>
  );
};

export default Footer;
