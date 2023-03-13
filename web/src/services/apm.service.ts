import trace from './mock/trace.data.json';
import trace2 from './mock/t.json';

//https://devicon.dev/
function getServiceList() {
  return [
    { name: 'order.service', type: 'go' },
    { name: 'user.service', type: 'java' },
    { name: 'api.gateway', type: 'rust' },
    { name: 'shop.service', type: 'php' },
    { name: 'ship.service', type: 'python' },
    { name: 'payment.service', type: 'dotnetcore' },
    { name: 'auth.service', type: 'nodejs' },
  ];
}

function getStats() {
  return {
    requests: Math.floor(Math.random() * 1000),
    avg: Math.floor(Math.random() * 2000),
    p95: Math.floor(Math.random() * 2000),
    errorRate: (Math.random() * 100).toFixed(2),
    alert: Math.floor(Math.random() * 5),
  };
}

function getServiceStats() {
  return [
    {
      tags: { type: 'go', name: 'order.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'java', name: 'user.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'rust', name: 'api.gateway' },
      fields: getStats(),
    },
    {
      tags: { type: 'php', name: 'shop.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'python', name: 'ship.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'dotnetcore', name: 'payment.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'nodejs', name: 'auth.service' },
      fields: getStats(),
    },
    {
      tags: { type: 'mysql', name: 'user.db' },
      fields: getStats(),
    },
    {
      tags: { type: 'apachekafka', name: 'order.queue' },
      fields: getStats(),
    },
  ];
}

function getTraceData() {
  return trace[0];
}

function getTraceData2(): any {
  return trace2;
}

export default {
  getServiceList,
  getServiceStats,
  getTraceData,
  getTraceData2,
};
