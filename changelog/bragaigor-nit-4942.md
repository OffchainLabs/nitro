### Changed
- `--node.transaction-streamer.shutdown-on-blockhash-mismatch` now defaults to `true`: on a feed-vs-local block hash mismatch the node refuses to persist or rebroadcast the result and shuts down gracefully.
