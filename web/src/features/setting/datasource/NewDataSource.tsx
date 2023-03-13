import React from 'react';
import { Card, List, Typography } from '@douyinfe/semi-ui';
import * as _ from 'lodash-es';
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
