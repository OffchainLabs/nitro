# Robinhood Nitro MEV Feed

This optional feed exports canonical Nitro blocks to a local consumer over a
Unix stream socket. It is a post-swap observation source, not a pending order
flow or transaction ordering interface.

## Enable

Create the socket directory before starting Nitro and configure:

```text
--execution.mev-feed.enable=true
--execution.mev-feed.required=true
--execution.mev-feed.socket-path=/run/robinhood-mev/feed.sock
--execution.mev-feed.queue-size=128
--execution.mev-feed.write-timeout=50ms
```

The node only sends data. A consumer cannot submit transactions or change
execution. With the default `enable=false`, no socket, worker, or queue is
created.

## Protocol

Each frame starts with the 24-byte big-endian header `RMEV`, version `1`, kind,
flags, per-session sequence, payload length, and CRC32C. Blocks are emitted as
`BLOCK_BEGIN`, zero or more `TRANSACTION`, then `BLOCK_END`. Transaction records
contain the signed RLP transaction, receipt fields, and all logs in transaction
order. A consumer must wait for `BLOCK_END` before applying a block.

`GAP` is sticky across reconnects and is emitted for queue overflow, disconnect,
write timeout, oversized or invalid records. `REORG` precedes a newly observed
block when its number or parent hash does not follow the previous canonical
cursor. After either condition the consumer must resync through RPC by block
hash before broadcasting.

Only one active consumer is accepted. Slow consumers are disconnected; feed
errors never propagate into Nitro block execution.
