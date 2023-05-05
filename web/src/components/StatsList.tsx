import { IconSearch } from '@douyinfe/semi-icons';
import { Card, Input, List } from '@douyinfe/semi-ui';
import { MetricSrv } from '@src/services';
import { TemplateKit } from '@src/utils';
import { useQuery } from '@tanstack/react-query';
import React from 'react';

const StatsList: React.FC<{ label: string }> = (props) => {
  const { label } = props;
  const { isLoading, data } = useQuery(['stats_list'], async () => {
    await new Promise((r) => setTimeout(r, 1000));
    return MetricSrv.getStatsList();
  });
  console.log(isLoading, data);

  return (
    <Card bodyStyle={{ padding: 0 }}>
      <List
        style={{ minHeight: 300 }}
        header={<Input style={{ width: '100%' }} placeholder="搜索" prefix={<IconSearch />} />}
        dataSource={data || []}
        renderItem={(item) => <List.Item>{TemplateKit.template(label, item.tags || {})}</List.Item>}
        loading={isLoading}
      />
    </Card>
  );
};

export default StatsList;
