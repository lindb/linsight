import { Button, Input, Modal, Radio, RadioGroup, Select, Typography } from '@douyinfe/semi-ui';
import { IconGridStroked, IconSearchStroked } from '@douyinfe/semi-icons';
import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { DashboardSrv } from '@src/services';
const { Title, Text } = Typography;

const AddToDashboard: React.FC<{ btnTheme?: any; btnType?: any }> = (props) => {
  const { btnType, btnTheme } = props;
  const [type, setType] = useState('1');
  const [visible, setVisible] = useState(false);
  const { data } = useQuery(['dashboard-list'], () => DashboardSrv.getDashboardList());

  console.log('add dashboard btn');

  const renderDashboard = () => {
    if (type === '1') {
      return <Input placeholder="Please input dashboard name" />;
    }
    return (
      <Select
        placeholder="Please select dashboard"
        filter
        showClear
        prefix={<IconSearchStroked />}
        style={{ width: '100%' }}>
        {(data || []).map((item: any) => (
          <Select.Option key={item.name} value={item.name}>
            {item.type && <i style={{ marginRight: 4 }} className={`devicon-${item.type}-plain colored`} />}
            {item.name}
          </Select.Option>
        ))}
      </Select>
    );
  };
  return (
    <>
      <Button
        type={btnType || 'tertiary'}
        theme={btnTheme || 'borderless'}
        icon={<IconGridStroked />}
        onClick={() => setVisible(true)}>
        Add to dashboard
      </Button>
      <Modal
        size="medium"
        title="Add metric to dashboard"
        className="x-monitor"
        visible={visible}
        motion={false}
        onCancel={() => setVisible(false)}
        footer={
          <>
            <Button type="tertiary" onClick={() => setVisible(false)}>
              Cancel
            </Button>
            <Button
              type="primary"
              theme="solid"
              icon={<IconGridStroked />}
              onClick={() => {
                setVisible(false);
                window.open(`${window.location.origin}/dashboard/view`);
              }}>
              Open Dashboard
            </Button>
          </>
        }>
        <div style={{ marginBottom: 4 }}>
          <Title heading={6}>Target dashboard</Title>
          <Text type="tertiary" size="small">
            Choose where to add the panel.
          </Text>
        </div>
        <RadioGroup type="button" defaultValue={'1'} onChange={(e: any) => setType(e.target.value)}>
          <Radio value="1">New Dashboard</Radio>
          <Radio value="2">Existing dashboard</Radio>
        </RadioGroup>
        <div style={{ width: '100%', marginTop: 16 }}>
          <div style={{ marginBottom: 4 }}>
            <Title heading={6}>Dashboard</Title>
          </div>
          {renderDashboard()}
        </div>
      </Modal>
    </>
  );
};

export default AddToDashboard;
