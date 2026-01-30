### Added
- Add config option `--node.transaction-streamer.broadcast-during-sync` to control broadcasting messages to feed clients during node synchronization. When false (default), only recent messages are broadcast during sync, preventing connected clients from being flooded with historical data.
