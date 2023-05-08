## Linsight

An open-source, all-in-one observability platform.

[![codecov](https://codecov.io/gh/lindb/linsight/branch/develop/graph/badge.svg)](https://codecov.io/gh/lindb/linsight)

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

