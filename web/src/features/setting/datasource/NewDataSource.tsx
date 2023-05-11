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
import { Card, List, Typography } from '@douyinfe/semi-ui';
import { DatasourcePlugin, DatasourceRepositoryInst } from '@src/types';
import { createSearchParams, useNavigate } from 'react-router-dom';

const { Title, Text } = Typography;

const NewDataSource: React.FC = () => {
  const navigate = useNavigate();

  const plugins = DatasourceRepositoryInst.getPlugins();

  const createDataSource = (type: string) => {
    navigate({ pathname: `/setting/datasource/edit`, search: `${createSearchParams({ type: type })}` });
  };

  return (
    <Card className="linsight-feature" bodyStyle={{ padding: '12px 0' }}>
      <List
        dataSource={plugins}
        renderItem={(item: DatasourcePlugin) => {
          return (
            <List.Item
              onClick={() => createDataSource(item.Type)}
              style={{ cursor: 'pointer' }}
              header={<img src={`${item.darkLogo}`} width={48} />}
              main={
                <div>
                  <Title heading={5}>{item.Name}</Title>
                  <div style={{ marginTop: 8 }}>
                    <Text>{item.Description}</Text>
                  </div>
                </div>
              }
            />
          );
        }}
      />
    </Card>
  );
};

export default NewDataSource;
