# Developer Notes

```shell
watchexec --exts go --watch . 'go test ./... && make lint'
```

uint64 type is based on [rollups_outputs.rs](https://github.com/cartesi/rollups-node/blob/392c75972037352ecf94fb482619781b1b09083f/offchain/rollups-events/src/rollups_outputs.rs#L41)

```go
Voucher
InputIndex  uint64
OutputIndex uint64
```

Input encoded by rollups-contract V2

```text
0xcc7dee1f000000000000000000000000000000000000000000000000cc0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007fa9385be102ac3eac297483dd6233d62b3e149600000000000000000000000000000000000000000000000000000000000000e10000000000000000000000000000000000000000000000000000000061d0c1b100000000000000000000000000000000000000000000000000000000000000e100000000000000000000000000000000000000000000000000000000000000e00000000000000000000000000000000000000000000000000000000000000020157f9f93806730d47e3a6e8b41b4fa9d98f89ec6f53627f3abf1d171769345eb
```

## How to run the HL GraphQL

Run the postgraphile

```bash
docker network create mynetwork
docker build -t postgresteste:latest ./postgres
docker run -d --network mynetwork -p 5432:5432 --name postgres postgresteste:latest

docker build -t postgraphile-custom ./postgraphile/
docker run -d --network mynetwork -p 5000:5000 --name postgraphile-custom postgraphile-custom
```

http://localhost:5000/graphiql

## Build 

```
go build
```


Run the nonodo with HL GraphQL flag enabled

```
./nonodo --high-level-graphql --enable-debug --node-version v2
```

```
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
./nonodo --http-address=0.0.0.0 --high-level-graphql --enable-debug --node-version v2 --db-implementation postgres
```

Disable sync

```
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
./nonodo --http-address=0.0.0.0 --high-level-graphql --enable-debug --node-version v2 --db-implementation postgres --graphile-disable-sync
```
