## Linsight

An open-source, all-in-one observability platform.

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

