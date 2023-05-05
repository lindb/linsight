import React, { useState } from 'react';
import { Chart } from '@src/components';
import { AlertSrv } from '@src/services';
import { Col, Row, Space, Typography, Card, Dropdown, Form } from '@douyinfe/semi-ui';
import {
  IconExit,
  IconSettingStroked,
  IconFullScreenStroked,
  IconBellStroked,
  IconEditStroked,
  IconSearchStroked,
  IconCopyStroked,
  IconLink,
  IconDownloadStroked,
} from '@douyinfe/semi-icons';
import { useNavigate } from '@src/hooks';
const { Text } = Typography;

const ChartCard: React.FC = () => {
  const [show, setShow] = useState(false);
  const navigate = useNavigate();
  return (
    <div onMouseMove={() => setShow(true)} onMouseOut={() => setShow(false)}>
      <Card
        bodyStyle={{ padding: 12 }}
        headerStyle={{ padding: 12 }}
        title={<>CPU Usage%</>}
        headerExtraContent={
          show && (
            <Space>
              <IconFullScreenStroked style={{ cursor: 'pointer' }} />
              <Dropdown
                render={
                  <Dropdown.Menu>
                    <Dropdown.Item>
                      <Text icon={<IconCopyStroked />}>Copy</Text>
                    </Dropdown.Item>
                    <Dropdown.Item>
                      <Text icon={<IconLink />}>Share Link</Text>
                    </Dropdown.Item>
                    <Dropdown.Item onClick={() => window.open(`${window.location.origin}/dashboard/explore`)}>
                      <Text icon={<IconSearchStroked />}>Explore</Text>
                    </Dropdown.Item>
                    <Dropdown.Item>
                      <Text icon={<IconDownloadStroked />}>Download as PNG</Text>
                    </Dropdown.Item>
                  </Dropdown.Menu>
                }>
                <IconExit rotate={270} style={{ cursor: 'pointer' }} />
              </Dropdown>
              <Dropdown
                render={
                  <Dropdown.Menu>
                    <Dropdown.Item onClick={() => navigate('/dashboard/edit')}>
                      <Text icon={<IconEditStroked />}>Edit</Text>
                    </Dropdown.Item>
                    <Dropdown.Item onClick={() => window.open(`${window.location.origin}/alert/config`)}>
                      <Text icon={<IconBellStroked />}>Alert</Text>
                    </Dropdown.Item>
                  </Dropdown.Menu>
                }>
                <IconSettingStroked style={{ cursor: 'pointer' }} />
              </Dropdown>
            </Space>
          )
        }>
        <Chart type="line" data={AlertSrv.getAlertStats()} height={260} />
      </Card>
    </div>
  );
};
const Dashboard: React.FC = () => {
  return (
    <div>
      <Card bodyStyle={{ padding: 0 }}>
        <Form style={{ marginLeft: 12 }}>
          <Form.Select
            label="Host"
            labelPosition="inset"
            field="datasource"
            multiple
            optionList={[
              { label: '192.168.0.1', value: '192.168.0.1' },
              { label: '192.168.0.2', value: '192.168.0.2' },
              { label: '192.168.0.3', value: '192.168.0.3' },
            ]}
          />
        </Form>
      </Card>
      <Row gutter={12} style={{ paddingTop: 12 }}>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
      </Row>
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
      </Row>
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
      </Row>
      <Row gutter={12} style={{ marginTop: 12 }}>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
        <Col span={8}>
          <ChartCard />
        </Col>
      </Row>
    </div>
  );
};

export default Dashboard;
