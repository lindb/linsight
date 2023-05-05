import { IconSearch } from '@douyinfe/semi-icons';
import { Select, Space } from '@douyinfe/semi-ui';
import { APMSrv } from '@src/services';
import { useQuery } from '@tanstack/react-query';
import React from 'react';

const ServiceFilter: React.FC = () => {
  const { data } = useQuery(['service_filter'], () => APMSrv.getServiceList());
  return (
    <div>
      <Select filter prefix={<IconSearch />} style={{ width: 270 }}>
        {(data || []).map((opt) => (
          <Select.Option key={opt.name}>
            <Space align="center">
              <i className={`devicon-${opt.type}-plain colored`} />
              <span>{opt.name}</span>
            </Space>
          </Select.Option>
        ))}
      </Select>
    </div>
  );
};

export default ServiceFilter;
