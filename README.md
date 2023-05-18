<p align="center">
  <img src="./web/src/images/logo_title.svg" alt="linsight-logo" height="72" />

  <p align="center">
	An open-source, comprehensive observability platform for monitoring your applications and troubleshooting issues, similar to DataDog, New Relic, Grafana and others.
  </p>
</p>

<div align="center">

[![codecov](https://codecov.io/gh/lindb/linsight/branch/develop/graph/badge.svg)](https://codecov.io/gh/lindb/linsight)

</div>

## Features

- **Unified Observability**: Linsight brings together logs, metrics, traces, and events in a single platform, providing a holistic view of your application's performance and health.
- **Distributed Tracing**: Gain deep insights into your application's performance by visualizing and analyzing traces across distributed systems.
- **Rich Visualization**: Create beautiful, interactive dashboards with a variety of visualization options, such as graphs, tables, lists, gauges, and more.
- **Alerting**: Set up custom alerts to notify you of potential issues in your applications, ensuring you can address problems before they impact your users.
- **Anomaly Detection**: Leverage advanced machine learning algorithms to automatically detect and highlight anomalies in your application performance data.
- **User-friendly Interface**: Linsight's intuitive UI makes it easy for both beginners and experienced users to navigate, configure, and manage their monitoring setup.

## Getting Started

### Docker

```sh
docker run --name=linsight -p 8080:8080 lindata/linsight
```

Open your browser and navigate to [http://localhost:8080](http://localhost:8080) to access the Linsight web interface.

## License

Linsight is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.

