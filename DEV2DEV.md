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
docker compose up --wait up postgraphile
```

[http://localhost:5001/graphiql](http://localhost:5001/graphiql)

### Build

```shell
go build
```

Run the nonodo with HL GraphQL flag enabled

```shell
go run . --high-level-graphql --enable-debug --node-version v2
```

```shell
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
go run . --http-address=0.0.0.0 --high-level-graphql --enable-debug --node-version v2 --db-implementation postgres
```

Disable sync

```shell
export POSTGRES_HOST=127.0.0.1
export POSTGRES_PORT=5432
export POSTGRES_DB=mydatabase
export POSTGRES_USER=myuser
export POSTGRES_PASSWORD=mypassword
go run . --http-address=0.0.0.0 --high-level-graphql --enable-debug --node-version v2 --db-implementation postgres --graphile-disable-sync
```

## Environment Variables

To configure the endpoint of the node v2 Graphile, you can set the GRAPHILE_URL environment variable. Here's how you can do it:

```bash
export GRAPHILE_URL=http://localhost:5001/graphql
```
