# Developer Notes

```shell
watchexec --exts go --watch . 'go test ./... && make lint'
```

uint64 type is based on [rollups_outputs.rs](https://github.com/cartesi/rollups-node/blob/392c75972037352ecf94fb482619781b1b09083f/offchain/rollups-events/src/rollups_outputs.rs#L41)

Voucher
InputIndex  uint64
OutputIndex uint64
