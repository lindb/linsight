function getAlertList() {
  return [
    { id: 1, status: 'OK', name: 'cpu usage>80%', tags: ['team:demo', 'env:prod', 'service:payment'], priority: 'P1' },
    {
      id: 2,
      status: 'ALERT',
      name: 'memory usage>80%',
      tags: ['team:demo', 'env:prod', 'service:order'],
      priority: 'P2',
    },
    { id: 3, status: 'OK', name: 'memory usage>80%', tags: ['team:demo', 'env:prod', 'service:order'], priority: 'P2' },
    { id: 4, status: 'ALERT', name: 'disk usage>80%', tags: ['team:demo', 'env:prod', 'service:user'], priority: 'P2' },
    {
      id: 5,
      status: 'ALERT',
      name: 'memory usage>80%',
      tags: ['team:demo', 'env:prod', 'service:order'],
      priority: 'P2',
    },
    { id: 6, status: 'OK', name: 'order count<100', tags: ['team:demo', 'env:prod', 'service:order'], priority: 'P1' },
    {
      id: 7,
      status: 'ALERT',
      name: 'memory usage>80%',
      tags: ['team:demo', 'env:prod', 'service:order'],
      priority: 'P2',
    },
  ];
}

function getAlertStats() {
  const dates = [];
  const dataset = [];
  for (let i = 0; i < 60; i++) {
    dates.push(i);
    if (i > 20 && i < 30) {
      dataset.push(null);
    } else {
      dataset.push(Math.floor(Math.random() * 200));
    }
  }

  return {
    labels: dates,
    datasets: [
      {
        label: 'Count',
        data: dataset,
      },
    ],
  };
}

function getAlertEvents() {
  return [
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'OK',
      color: 'success',
      title: '[Recovered] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'WARN',
      color: 'warning',
      title: '[Warn] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'TRIGGER',
      color: 'danger',
      title: '[Recovered] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'OK',
      color: 'success',
      title: '[Recovered] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'OK',
      color: 'success',
      title: '[Recovered] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
    {
      date: 'Sun, Nov 27, 4:59:18 pm',
      status: 'OK',
      color: 'success',
      title: '[Recovered] CPU Load is high on 192.168.0.1',
      content: 'system.cpu.user over host:192.168.0.1 was <= 3.0 at all times during the last 5m.',
      last: 'The monitor was last triggered at Sun Nov 27 2022 08:53:08 UTC.',
      notify: 'Notify:  @demo@google.com, please know this.',
    },
  ];
}

export default {
  getAlertList,
  getAlertStats,
  getAlertEvents,
};
