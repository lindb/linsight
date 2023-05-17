<p align="center">
  <img src="./web/src/images/logo_title.svg" alt="linsight-logo" height="72" />

  <p align="center">
	An open-source, comprehensive observability platform for monitoring your applications and troubleshooting issues, similar to DataDog, New Relic, and others.
  </p>
</p>

<div align="center">

[![codecov](https://codecov.io/gh/lindb/linsight/branch/develop/graph/badge.svg)](https://codecov.io/gh/lindb/linsight)

</div>

- init db records

```sh
go run github.com/lindb/linsight/cmd migrator all
```

- start backend server

```sh
make run
```

- web debug

1. node >= 18

```sh 
cd web
yarn install 
yarn dev
```

